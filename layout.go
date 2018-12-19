// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 29 Nov 2018
//  FILE: layout.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines overall layout and management of UI widgets, keypress event
//    handlers, and other high-level user interactions.
//
// =============================================================================

package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"bytes"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	sideColumnWidth = 32
	logRowsHeight   = 5 // number of visible log lines + 1
)

var (
	idleUpdateFreq time.Duration = 1 * time.Second
	busyUpdateFreq time.Duration = 100 * time.Millisecond
)

var (
	colorScheme = struct {
		inactiveText     tcell.Color
		activeText       tcell.Color
		inactiveMenuText tcell.Color
		activeMenuText   tcell.Color
		inactiveBorder   tcell.Color
		activeBorder     tcell.Color
		clockText        tcell.Color
		statusText       tcell.Color
		statusIndicator  tcell.Color
		identityText     tcell.Color
	}{
		inactiveText:     tcell.ColorGray,
		activeText:       tcell.ColorLightSlateGray,
		inactiveMenuText: tcell.ColorSkyblue,
		activeMenuText:   tcell.ColorDodgerBlue,
		inactiveBorder:   tcell.ColorWhite,
		activeBorder:     tcell.ColorSkyblue,
		clockText:        tcell.ColorDodgerBlue,
		statusText:       tcell.ColorGreenYellow,
		statusIndicator:  tcell.ColorDarkOrange,
		identityText:     tcell.ColorGreenYellow,
	}
)

// type BusyState keeps track of the number of goroutines that are wishing to
// indicate to the UI that they are active or busy, that the user should hold
// their horses.
type BusyState struct {
	changed   chan uint64 // signal when busy state changes
	_         uintptr     // padding, 64-bit atomic ops must be performed on 8-byte boundaries (see go1.10 sync/atomic bugs)
	busyCount uint64      // number of busy goroutines
	busyCycle uint64      // number of UI updates performed while busy
}

// function newBusyState() instantiates a new BusyState object with zeroized
// counter and update cycle.
func newBusyState() *BusyState {
	return &BusyState{
		changed:   make(chan uint64),
		busyCount: 0,
		busyCycle: 0,
	}
}

// function count() safely returns the current number of goroutines currently
// declaring themselves as busy.
func (s *BusyState) count() int {
	count := atomic.LoadUint64(&s.busyCount)
	return int(count)
}

// function inc() safely increments the number of goroutines currently declaring
// themselves as busy by 1.
func (s *BusyState) inc() int {
	newCount := atomic.AddUint64(&s.busyCount, 1)
	s.changed <- newCount
	// reset the cycle if we were not busy before this increment
	if 1 == newCount {
		s.reset()
	}
	return int(newCount)
}

// function dec() safely decrements the number of goroutines currently declaring
// themselves as busy by 1.
func (s *BusyState) dec() int {
	newCount := atomic.AddUint64(&s.busyCount, ^uint64(0))
	s.changed <- newCount
	// reset the cycle if we are not busy after this increment
	if 0 == newCount {
		s.reset()
	}
	return int(newCount)
}

// function count() safely returns the current number of goroutines currently
// declaring themselves as busy.
func (s *BusyState) cycle() int {
	cycle := atomic.LoadUint64(&s.busyCycle)
	return int(cycle)
}

// function next() safely increments by 1 the UI cycles elapsed since the
// current busy state was initiated.
func (s *BusyState) next() int {
	cycle := atomic.AddUint64(&s.busyCycle, 1)
	return int(cycle)
}

// function reset() safely resets the current UI cycles elapsed to 0.
func (s *BusyState) reset() {
	atomic.StoreUint64(&s.busyCycle, 0)
}

// type Layout holds the high level components of the terminal user interface
// as well as the main tview runtime API object tview.Application.
type Layout struct {
	ui     *tview.Application
	option *Options
	busy   *BusyState

	pages     *tview.Pages
	pagesRoot string

	root *tview.Grid
	main *tview.Box

	logView   *LogView
	libSelect *LibSelectView
	helpInfo  *HelpInfoView

	focusQueue chan FocusDelegator
	focused    FocusDelegator

	// NOTE: this screen var won't get set until one of the draw routines which
	// uses it is called, so be careful when accessing it -- make sure it's
	// actually available.
	screen *tcell.Screen
}

// function show() starts drawing the user interface.
func (l *Layout) show() *ReturnCode {

	// timer forcing the app to redraw any areas that may have changed. this
	// update frequency is dynamic -- more frequent while the "Busy" indicator
	// is active, less frequent while it isn't.
	go func(l *Layout) {

		// use the CPU-intensive frequency by default to err on the side of
		// caution.
		updateFreq := busyUpdateFreq

		setFreq := func(curr, freq *time.Duration) bool {
			if *curr != *freq {
				*curr = *freq
				return true
			}
			return false
		}

		for {
			tick := time.NewTicker(updateFreq)
		REFRESH:
			for {
				select {
				case <-tick.C:
					l.ui.QueueUpdateDraw(func() {})
				case count := <-l.busy.changed:
					l.ui.QueueUpdateDraw(func() {})
					// use ifFreq() so that we kill the Ticker and alloc a new
					// one if and only if the duration actually changed.
					switch count {
					case 0:
						if setFreq(&updateFreq, &idleUpdateFreq) {
							break REFRESH
						}
					default:
						if setFreq(&updateFreq, &busyUpdateFreq) {
							break REFRESH
						}
					}
				}
			}
			tick.Stop()
		}
	}(l)

	// signal monitor for refocus requests. when an event occurs that requires a
	// new widget to be focused, this routine will call the interface-compliant
	// widgets' event handlers to blur() and focus() the old and new widgets,
	// respectively.
	go func(l *Layout) {
		for {
			select {
			case delegate := <-l.focusQueue:
				if delegate != nil {
					if delegate != l.focused {
						if l.focused != nil {
							l.focused.blur()
						}
						delegate.focus()
						// update afterwards so that the focus() method can make
						// decisions based on which was previously focused.
						l.focused = delegate
					}
					l.ui.Draw()
				}
			}
		}
	}(l)

	l.focusQueue <- l.logView

	if err := l.ui.Run(); err != nil {
		return rcTUIError.specf("show(): ui.Run(): %s", err)
	}
	return nil
}

// function newLayout() creates the initial layout of the user interface and
// populates it with the primary widgets. each Library passed in as argument
// has its Layout field initialized with this instance.
func newLayout(opt *Options, busy *BusyState, lib ...*Library) *Layout {

	var layout Layout

	ui := tview.NewApplication()

	header := tview.NewBox().
		SetBorder(false)

	main := tview.NewBox().
		SetBorder(false)

	logView := newLogView(ui, "root")

	footer := tview.NewBox().
		SetBorder(false)

	root := tview.NewGrid().
		// these are actual sizes, in terms of addressable terminal locations,
		// i.e. characters and lines. the literal width and height values in the
		// arguments to AddItem() are the logical sizes, in terms of rows and
		// columns that are laid out by the arguments to SetRows()/SetColumns().
		SetRows(1, 0, logRowsHeight, 1).
		SetColumns(sideColumnWidth, 0, sideColumnWidth).
		// fixed components that are always visible
		AddItem(header /****/, 0, 0, 1, 3, 0, 0, false).
		AddItem(main /******/, 1, 0, 1, 3, 0, 0, false).
		AddItem(logView /***/, 2, 0, 1, 3, 0, 0, false).
		AddItem(footer /****/, 3, 0, 1, 3, 0, 0, false)

	root. // other options for the primary layout grid
		SetBorders(true).
		SetBorderColor(colorScheme.inactiveBorder)

	libSelect := newLibSelectView(ui, "libSelect")

	helpInfo := newHelpInfoView(ui, "helpInfo")

	pages := tview.NewPages().
		AddPage("root", root, true, true).
		AddPage(libSelect.page(), libSelect, false, true).
		AddPage(helpInfo.page(), helpInfo, false, true)

	header. // register the header bar screen drawing callback
		SetDrawFunc(layout.drawMenuBar)

	footer. // register the status bar screen drawing callback
		SetDrawFunc(layout.drawStatusBar)

	// define the higher-order tab cycle
	logView.setDelegates(&layout, nil, nil)
	libSelect.setDelegates(&layout, logView, logView)
	helpInfo.setDelegates(&layout, nil, nil)

	// and finally initialize our actual Layout object to be returned
	layout = Layout{
		ui:     ui,
		option: opt,
		busy:   busy,

		pages:     pages,
		pagesRoot: "root",

		root: root,
		main: main,

		logView:   logView,
		libSelect: libSelect,
		helpInfo:  helpInfo,

		focusQueue: make(chan FocusDelegator),
		focused:    nil,

		screen: nil,
	}

	// add a ref to this layout object to all libraries
	for _, l := range lib {
		l.layout = &layout
	}

	// set the initial page displayed when application begins
	pages.SwitchToPage(layout.pagesRoot)

	ui. // global tview application configuration
		SetRoot(pages, true).
		SetInputCapture(layout.inputEvent)

	return &layout
}

// function isGlobalInputEvent() determines if the program is in a state to
// accept global input keys. for example, if the program is currently presenting
// a modal window or dialog, then we don't necessarily want to allow the global
// cycle-focused-view event key.
func (l *Layout) isGlobalInputEvent(event *tcell.EventKey) bool {
	return true
}

func (l *Layout) shouldDelegateInputEvent(event *tcell.EventKey) bool {

	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'h', 'H', 'j', 'J', 'k', 'K', 'l', 'L':
			// do NOT support the vi-style navigation keys in the log view
			if l.focused == l.logView {
				return false
			}
		}
	}
	return true
}

// function inputEvent() is the application-level keyboard input event handler.
// this function provides an opportunity to override or reject input keys before
// ever forwarding them onto the focused view. it also defines the global key
// event handlers such as for cycling focus among the available views.
func (l *Layout) inputEvent(event *tcell.EventKey) *tcell.EventKey {

	fwdEvent := event

	if l.isGlobalInputEvent(event) {

		switch event.Key() {
		case tcell.KeyCtrlC:
			// don't exit on Ctrl+C, it feels unsanitary. instead, notify the
			// user we can exit cleanly by simply pressing 'q'.
			fwdEvent = nil

		case tcell.KeyEsc:
			l.focusQueue <- l.logView

		case tcell.KeyRune:
			switch event.Rune() {
			case 'l', 'L':
				l.focusQueue <- l.libSelect
			case 'h', 'H':
				l.focusQueue <- l.helpInfo
			case 'q', 'Q':
				l.ui.Stop()
			}

			// TODO: remove me, exists only for eval of color palettes
			if fn, ok := logColors[event.Rune()]; ok {
				fn()
			}

		case tcell.KeyTab:
			if nil != l.focused {
				next := l.focused.next()
				if nil != next {
					l.focusQueue <- next
				}
			}
		}

		if !l.shouldDelegateInputEvent(event) {
			fwdEvent = nil
		}
	}

	return fwdEvent
}

// function drawMenuBar() is the callback handler associated with the top-most
// header box. this routine is not called on-demand, but is usually invoked
// implicitly by other re-draw events.
func (l *Layout) drawMenuBar(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	const (
		libDimWidth   = 40 // library selection window width
		libDimHeight  = 10 // ^----------------------- height
		helpDimWidth  = 40 // help info window width
		helpDimHeight = 10 // ^--------------- height
	)

	// update the layout's associated screen field. note that you must be very
	// careful and not access this field until this status line has been drawn
	// at least one time.
	if nil == l.screen {
		l.screen = &screen
	}

	l.libSelect.
		SetRect(2, 1, libDimWidth, libDimHeight)

	l.helpInfo.
		SetRect(width-helpDimWidth, 1, helpDimWidth, helpDimHeight)

	library := fmt.Sprintf("[::bu]%s[::-]%s:", "L", "ibrary")
	help := fmt.Sprintf("[::bu]%s[::-]%s", "H", "elp")

	tview.Print(screen, library, x+3, y, width, tview.AlignLeft, colorScheme.inactiveMenuText)
	tview.Print(screen, help, x, y, width-3, tview.AlignRight, colorScheme.inactiveMenuText)

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

// function drawStatusBar() is the callback handler associated with the bottom-
// most footer box. this routine is regularly called so that the datetime clock
// remains accurate along with any status information currently available.
func (l *Layout) drawStatusBar(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	// the number of periods to draw incrementally during the "working"
	// animation is equal to: ellipses - 1
	const ellipses = 4

	// update the layout's associated screen field. note that you must be very
	// careful and not access this field until this status line has been drawn
	// at least one time.
	if nil == l.screen {
		l.screen = &screen
	}

	//dateTime := time.Now().Format("[15:04:05] Monday, January 02, 2006")
	dateTime := time.Now().Format("2006/01/02 15:04:05")

	// Write some text along the horizontal line.
	tview.Print(screen, dateTime, x+3, y, width, tview.AlignLeft, colorScheme.clockText)

	// update the busy indicator if we have any active worker threads
	count := l.busy.count()
	if count > 0 {
		// increment the screen refresh counter
		cycle := l.busy.next()

		// draw the "working..." indicator. note the +2 is to make room for the
		// moon rune following this indicator.
		working := fmt.Sprintf("working%-*s", ellipses, bytes.Repeat([]byte{'.'}, cycle%ellipses))
		tview.Print(screen, working, x-ellipses+1, y, width, tview.AlignRight, colorScheme.statusText)

		// draw the cyclic moon rotation
		moon := fmt.Sprintf("%c ", MoonPhase[cycle%MoonPhaseLength])
		tview.Print(screen, moon, x, y, width, tview.AlignRight, colorScheme.statusIndicator)
	}

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

// type View defines the related fields describing any visual widget or
// component displayed in a Layout.
type FocusDelegator interface {
	page() string
	next() FocusDelegator
	prev() FocusDelegator
	focus()
	blur()
}

type HelpInfoView struct {
	*tview.Box
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newHelpInfoView() allocates and initializes the tview.Form widget
// where the user selects which library to browse and any other filtering
// options.
func newHelpInfoView(ui *tview.Application, page string) *HelpInfoView {

	view := tview.NewBox()

	view.
		SetBorder(true).
		SetBorderColor(colorScheme.activeBorder).
		SetTitle(" Help ").
		SetTitleColor(colorScheme.activeMenuText).
		SetTitleAlign(tview.AlignRight)

	v := HelpInfoView{view, nil, page, nil, nil}

	view.SetDrawFunc(v.drawHelpInfoView)

	return &v
}

func (v *HelpInfoView) setDelegates(layout *Layout, prev, next FocusDelegator) {
	v.layout = layout
	v.focusPrev = prev
	v.focusNext = next
}
func (v *HelpInfoView) page() string         { return v.focusPage }
func (v *HelpInfoView) next() FocusDelegator { return v.focusNext }
func (v *HelpInfoView) prev() FocusDelegator { return v.focusPrev }
func (v *HelpInfoView) focus() {
	page := v.page()
	v.layout.pages.ShowPage(page)
}
func (v *HelpInfoView) blur() {
	page := v.page()
	v.layout.pages.HidePage(page)
}
func (v *HelpInfoView) drawHelpInfoView(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	swvers := fmt.Sprintf(" %s v%s (%s) ", identity, version, revision)

	tview.Print(screen, swvers, x+1, y, width-2, tview.AlignLeft, colorScheme.identityText)

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

type LibSelectView struct {
	*tview.Form
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newLibSelectView() allocates and initializes the tview.Form widget
// where the user selects which library to browse and any other filtering
// options.
func newLibSelectView(ui *tview.Application, page string) *LibSelectView {

	view := tview.NewForm()

	view.
		SetBorder(true).
		SetBorderColor(colorScheme.activeBorder).
		SetTitle(" Library ").
		SetTitleColor(colorScheme.activeMenuText).
		SetTitleAlign(tview.AlignLeft)

	v := LibSelectView{view, nil, page, nil, nil}

	return &v
}

func (v *LibSelectView) setDelegates(layout *Layout, prev, next FocusDelegator) {
	v.layout = layout
	v.focusPrev = prev
	v.focusNext = next
}
func (v *LibSelectView) page() string         { return v.focusPage }
func (v *LibSelectView) next() FocusDelegator { return v.focusNext }
func (v *LibSelectView) prev() FocusDelegator { return v.focusPrev }
func (v *LibSelectView) focus() {
	page := v.page()
	v.layout.pages.ShowPage(page)
}
func (v *LibSelectView) blur() {
	page := v.page()
	v.layout.pages.HidePage(page)
}

type LogView struct {
	*tview.TextView
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newLogView() allocates and initializes the tview.TextView widget
// where all runtime log data is navigated by and displayed to the user.
func newLogView(ui *tview.Application, page string) *LogView {

	logChanged := func() { ui.QueueUpdateDraw(func() {}) }
	logDone := func(key tcell.Key) {}

	view := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(colorScheme.activeText).
		SetWordWrap(true).
		SetWrap(false)

	view. // update the TextView event handlers
		SetChangedFunc(logChanged).
		SetDoneFunc(logDone).
		SetBorder(false)

	v := LogView{view, nil, page, nil, nil}

	return &v
}

func (v *LogView) setDelegates(layout *Layout, prev, next FocusDelegator) {
	v.layout = layout
	v.focusPrev = prev
	v.focusNext = next
}
func (v *LogView) page() string         { return v.focusPage }
func (v *LogView) next() FocusDelegator { return v.focusNext }
func (v *LogView) prev() FocusDelegator { return v.focusPrev }
func (v *LogView) focus() {
	page := v.page()
	v.layout.pages.ShowPage(page)
	v.layout.ui.SetFocus(v.TextView)
}
func (v *LogView) blur() {
}

// -----------------------------------------------------------------------------
//  temporary code below while evaluating color palettes
// -----------------------------------------------------------------------------

var logColors = map[rune]func(){

	'1': func() {
		infoLog.log("[#000000]ColorBlack")
		infoLog.log("[#000080]ColorNavy")
		infoLog.log("[#00008b]ColorDarkBlue")
		infoLog.log("[#0000cd]ColorMediumBlue")
		infoLog.log("[#0000ff]ColorBlue")
		infoLog.log("[#006400]ColorDarkGreen")
		infoLog.log("[#008000]ColorGreen")
		infoLog.log("[#008080]ColorTeal")
		infoLog.log("[#008b8b]ColorDarkCyan")
		infoLog.log("[#00bfff]ColorDeepSkyBlue")
		infoLog.log("[#00ced1]ColorDarkTurquoise")
		infoLog.log("[#00fa9a]ColorMediumSpringGreen")
		infoLog.log("[#00ff00]ColorLime")
		infoLog.log("[#00ff7f]ColorSpringGreen")
		infoLog.log("[#00ffff]ColorAqua")
		infoLog.log("[#191970]ColorMidnightBlue")
		infoLog.log("[#1e90ff]ColorDodgerBlue")
		infoLog.log("[#20b2aa]ColorLightSeaGreen")
		infoLog.log("[#228b22]ColorForestGreen")
		infoLog.log("[#2e8b57]ColorSeaGreen")
	},

	'2': func() {
		infoLog.log("[#2f4f4f]ColorDarkSlateGray")
		infoLog.log("[#32cd32]ColorLimeGreen")
		infoLog.log("[#3cb371]ColorMediumSeaGreen")
		infoLog.log("[#40e0d0]ColorTurquoise")
		infoLog.log("[#4169e1]ColorRoyalBlue")
		infoLog.log("[#4682b4]ColorSteelBlue")
		infoLog.log("[#483d8b]ColorDarkSlateBlue")
		infoLog.log("[#48d1cc]ColorMediumTurquoise")
		infoLog.log("[#4b0082]ColorIndigo")
		infoLog.log("[#556b2f]ColorDarkOliveGreen")
		infoLog.log("[#5f9ea0]ColorCadetBlue")
		infoLog.log("[#6495ed]ColorCornflowerBlue")
		infoLog.log("[#663399]ColorRebeccaPurple")
		infoLog.log("[#66cdaa]ColorMediumAquamarine")
		infoLog.log("[#696969]ColorDimGray")
		infoLog.log("[#6a5acd]ColorSlateBlue")
		infoLog.log("[#6b8e23]ColorOliveDrab")
		infoLog.log("[#708090]ColorSlateGray")
		infoLog.log("[#778899]ColorLightSlateGray")
		infoLog.log("[#7b68ee]ColorMediumSlateBlue")
	},

	'3': func() {
		infoLog.log("[#7cfc00]ColorLawnGreen")
		infoLog.log("[#7fff00]ColorChartreuse")
		infoLog.log("[#7fffd4]ColorAquaMarine")
		infoLog.log("[#800000]ColorMaroon")
		infoLog.log("[#800080]ColorPurple")
		infoLog.log("[#808000]ColorOlive")
		infoLog.log("[#808080]ColorGray")
		infoLog.log("[#87ceeb]ColorSkyblue")
		infoLog.log("[#87cefa]ColorLightSkyBlue")
		infoLog.log("[#8a2be2]ColorBlueViolet")
		infoLog.log("[#8b0000]ColorDarkRed")
		infoLog.log("[#8b008b]ColorDarkMagenta")
		infoLog.log("[#8b4513]ColorSaddleBrown")
		infoLog.log("[#8fbc8f]ColorDarkSeaGreen")
		infoLog.log("[#90ee90]ColorLightGreen")
		infoLog.log("[#9370db]ColorMediumPurple")
		infoLog.log("[#9400d3]ColorDarkViolet")
		infoLog.log("[#98fb98]ColorPaleGreen")
		infoLog.log("[#9932cc]ColorDarkOrchid")
		infoLog.log("[#9acd32]ColorYellowGreen")
	},

	'4': func() {
		infoLog.log("[#a0522d]ColorSienna")
		infoLog.log("[#a52a2a]ColorBrown")
		infoLog.log("[#a9a9a9]ColorDarkGray")
		infoLog.log("[#add8e6]ColorLightBlue")
		infoLog.log("[#adff2f]ColorGreenYellow")
		infoLog.log("[#afeeee]ColorPaleTurquoise")
		infoLog.log("[#b0c4de]ColorLightSteelBlue")
		infoLog.log("[#b0e0e6]ColorPowderBlue")
		infoLog.log("[#b22222]ColorFireBrick")
		infoLog.log("[#b8860b]ColorDarkGoldenrod")
		infoLog.log("[#ba55d3]ColorMediumOrchid")
		infoLog.log("[#bc8f8f]ColorRosyBrown")
		infoLog.log("[#bdb76b]ColorDarkKhaki")
		infoLog.log("[#c0c0c0]ColorSilver")
		infoLog.log("[#c71585]ColorMediumVioletRed")
		infoLog.log("[#cd5c5c]ColorIndianRed")
		infoLog.log("[#cd853f]ColorPeru")
		infoLog.log("[#d2691e]ColorChocolate")
		infoLog.log("[#d2b48c]ColorTan")
		infoLog.log("[#d3d3d3]ColorLightGray")
	},

	'5': func() {
		infoLog.log("[#d8bfd8]ColorThistle")
		infoLog.log("[#da70d6]ColorOrchid")
		infoLog.log("[#daa520]ColorGoldenrod")
		infoLog.log("[#db7093]ColorPaleVioletRed")
		infoLog.log("[#dc143c]ColorCrimson")
		infoLog.log("[#dcdcdc]ColorGainsboro")
		infoLog.log("[#dda0dd]ColorPlum")
		infoLog.log("[#deb887]ColorBurlyWood")
		infoLog.log("[#e0ffff]ColorLightCyan")
		infoLog.log("[#e6e6fa]ColorLavender")
		infoLog.log("[#e9967a]ColorDarkSalmon")
		infoLog.log("[#ee82ee]ColorViolet")
		infoLog.log("[#eee8aa]ColorPaleGoldenrod")
		infoLog.log("[#f08080]ColorLightCoral")
		infoLog.log("[#f0e68c]ColorKhaki")
		infoLog.log("[#f0f8ff]ColorAliceBlue")
		infoLog.log("[#f0fff0]ColorHoneydew")
		infoLog.log("[#f0ffff]ColorAzure")
		infoLog.log("[#f4a460]ColorSandyBrown")
		infoLog.log("[#f5deb3]ColorWheat")
	},

	'6': func() {
		infoLog.log("[#f5f5dc]ColorBeige")
		infoLog.log("[#f5f5f5]ColorWhiteSmoke")
		infoLog.log("[#f5fffa]ColorMintCream")
		infoLog.log("[#f8f8ff]ColorGhostWhite")
		infoLog.log("[#fa8072]ColorSalmon")
		infoLog.log("[#faebd7]ColorAntiqueWhite")
		infoLog.log("[#faf0e6]ColorLinen")
		infoLog.log("[#fafad2]ColorLightGoldenrodYellow")
		infoLog.log("[#fdf5e6]ColorOldLace")
		infoLog.log("[#ff0000]ColorRed")
		infoLog.log("[#ff00ff]ColorFuchsia")
		infoLog.log("[#ff1493]ColorDeepPink")
		infoLog.log("[#ff4500]ColorOrangeRed")
		infoLog.log("[#ff6347]ColorTomato")
		infoLog.log("[#ff69b4]ColorHotPink")
		infoLog.log("[#ff7f50]ColorCoral")
		infoLog.log("[#ff8c00]ColorDarkOrange")
		infoLog.log("[#ffa07a]ColorLightSalmon")
		infoLog.log("[#ffa500]ColorOrange")
		infoLog.log("[#ffb6c1]ColorLightPink")
	},

	'7': func() {
		infoLog.log("[#ffc0cb]ColorPink")
		infoLog.log("[#ffd700]ColorGold")
		infoLog.log("[#ffdab9]ColorPeachPuff")
		infoLog.log("[#ffdead]ColorNavajoWhite")
		infoLog.log("[#ffe4b5]ColorMoccasin")
		infoLog.log("[#ffe4c4]ColorBisque")
		infoLog.log("[#ffe4e1]ColorMistyRose")
		infoLog.log("[#ffebcd]ColorBlanchedAlmond")
		infoLog.log("[#ffefd5]ColorPapayaWhip")
		infoLog.log("[#fff0f5]ColorLavenderBlush")
		infoLog.log("[#fff5ee]ColorSeashell")
		infoLog.log("[#fff8dc]ColorCornsilk")
		infoLog.log("[#fffacd]ColorLemonChiffon")
		infoLog.log("[#fffaf0]ColorFloralWhite")
		infoLog.log("[#fffafa]ColorSnow")
		infoLog.log("[#ffff00]ColorYellow")
		infoLog.log("[#ffffe0]ColorLightYellow")
		infoLog.log("[#fffff0]ColorIvory")
		infoLog.log("[#ffffff]ColorWhite")
	},
}
