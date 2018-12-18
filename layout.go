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
	logRowsHeight   = 4
)

var (
	idleUpdateFreq time.Duration = 1 * time.Second
	busyUpdateFreq time.Duration = 100 * time.Millisecond
)

var (
	colorScheme = map[string]*tcell.Color{
		"primitive-background-color":     &tview.Styles.PrimitiveBackgroundColor,
		"contrast-background-color":      &tview.Styles.ContrastBackgroundColor,
		"more-contrast-background-color": &tview.Styles.MoreContrastBackgroundColor,
		"border-color":                   &tview.Styles.BorderColor,
		"title-color":                    &tview.Styles.TitleColor,
		"graphics-color":                 &tview.Styles.GraphicsColor,
		"primary-text-color":             &tview.Styles.PrimaryTextColor,
		"secondary-text-color":           &tview.Styles.SecondaryTextColor,
		"tertiary-text-color":            &tview.Styles.TertiaryTextColor,
		"inverse-text-color":             &tview.Styles.InverseTextColor,
		"contrast-secondary-text-color":  &tview.Styles.ContrastSecondaryTextColor,
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

	grid *tview.Grid
	main *tview.Box
	log  *LogView

	sel *LibSelectView

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

	l.focusQueue <- l.log

	if err := l.ui.Run(); err != nil {
		return rcTUIError.specf("show(): ui.Run(): %s", err)
	}
	return nil
}

// function newLayout() creates the initial layout of the user interface and
// populates it with the primary widgets. each Library passed in as argument
// has its Layout field initialized with this instance.
func newLayout(opt *Options, busy *BusyState, lib ...*Library) *Layout {

	ui := tview.NewApplication()

	header := tview.NewBox().
		SetBorder(false)

	main := tview.NewBox().
		SetBorder(false)

	log := newLogView(ui, "grid")

	footer := tview.NewBox().
		SetBorder(false)

	grid := tview.NewGrid().
		// these are actual sizes, in terms of addressable terminal locations,
		// i.e. characters and lines. the literal width and height values in the
		// arguments to AddItem() are the logical sizes, in terms of rows and
		// columns that are laid out by the arguments to SetRows()/SetColumns().
		SetRows(1, 0, logRowsHeight, 1).
		SetColumns(sideColumnWidth, 0, sideColumnWidth).
		// fixed components that are always visible
		AddItem(header /****/, 0, 0, 1, 3, 0, 0, false).
		//
		AddItem(main /******/, 1, 0, 1, 3, 0, 0, false).
		//
		AddItem(log /*******/, 2, 0, 1, 3, 0, 0, false).
		AddItem(footer /****/, 3, 0, 1, 3, 0, 0, false)

	grid. // other options for the primary layout grid
		SetBorders(true)

	sel := newLibSelectView(ui, "sel")

	pages := tview.NewPages().
		AddPage("grid", grid, true, true).
		AddPage("sel", sel, false, true)

	layout := Layout{
		ui:         ui,
		option:     opt,
		busy:       busy,
		pages:      pages,
		pagesRoot:  "grid",
		grid:       grid,
		main:       main,
		log:        log,
		sel:        sel,
		focusQueue: make(chan FocusDelegator),
		focused:    nil,
		screen:     nil,
	}

	// add a ref to this layout object to all libraries
	for _, l := range lib {
		l.layout = &layout
	}

	pages.SwitchToPage(layout.pagesRoot)

	// define the higher-order tab cycle
	log.setDelegates(&layout, sel, sel)
	sel.setDelegates(&layout, log, log)

	header.
		SetDrawFunc(layout.drawMenuBar)

	footer. // register the status bar screen drawing callback
		SetDrawFunc(layout.drawStatus)

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
			if l.focused == l.log {
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

		case tcell.KeyEsc, tcell.KeyEnter:
			l.focusQueue <- l.log

		case tcell.KeyRune:
			switch event.Rune() {
			case 'l', 'L':
				l.focusQueue <- l.sel
			case 'q', 'Q':
				l.ui.Stop()
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

func (l *Layout) drawMenuBar(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	// update the layout's associated screen field. note that you must be very
	// careful and not access this field until this status line has been drawn
	// at least one time.
	if nil == l.screen {
		l.screen = &screen
	}

	library := fmt.Sprintf("[::bu]%s[::-]%s:", "L", "ibrary")
	help := fmt.Sprintf("[::bu]%s[::-]%s", "H", "elp")

	tview.Print(screen, library, x+3, y, width, tview.AlignLeft, tview.Styles.PrimaryTextColor)
	tview.Print(screen, help, x, y, width-3, tview.AlignRight, tview.Styles.PrimaryTextColor)

	return 0, 0, 0, 0
}

// function drawStatus() is the callback handler associated with the bottom-most
// footer box. this routine is regularly called so that the datetime clock
// remains accurate along with any status information currently available.
func (l *Layout) drawStatus(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

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
	tview.Print(screen, dateTime, x+3, y, width, tview.AlignLeft, tcell.ColorGreen)

	// update the busy indicator if we have any active worker threads
	count := l.busy.count()
	if count > 0 {
		// increment the screen refresh counter
		cycle := l.busy.next()

		// draw the "working..." indicator. note the +2 is to make room for the
		// moon rune following this indicator.
		working := fmt.Sprintf("working%-*s", ellipses, bytes.Repeat([]byte{'.'}, cycle%ellipses))
		tview.Print(screen, working, x-ellipses+1, y, width, tview.AlignRight, tcell.ColorAqua)

		// draw the cyclic moon rotation
		moon := fmt.Sprintf("%c ", MoonPhase[cycle%MoonPhaseLength])
		tview.Print(screen, moon, x, y, width, tview.AlignRight, tcell.ColorDarkOrange)
	}

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

// type View defines the related fields describing any visual widget or
// component displayed in a Layout.
type FocusDelegator interface {
	next() FocusDelegator
	prev() FocusDelegator
	focus()
	blur()
	page() string
}

type LibSelectView struct {
	*tview.Form
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newLogView() allocates and initializes the tview.Form widget where
// the user selects which library to browse and any other filtering options.
func newLibSelectView(ui *tview.Application, page string) *LibSelectView {

	form := tview.NewForm()

	form.
		SetBorder(true).
		SetTitle("Library").
		SetTitleAlign(tview.AlignLeft)

	form.
		SetRect(3, 1, 40, 10)

	return &LibSelectView{form, nil, page, nil, nil}
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

	log := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGhostWhite).
		SetWordWrap(true).
		SetWrap(false)

	log. // update the TextView event handlers
		SetChangedFunc(logChanged).
		SetDoneFunc(logDone).
		SetBorder(false)

	return &LogView{log, nil, page, nil, nil}
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
