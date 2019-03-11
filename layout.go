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
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	sideColumnWidth = 32
	logRowsHeight   = 6 // number of visible log lines + 1
)

//var (
//	idleUpdateFreq time.Duration = 30 * time.Second
//	busyUpdateFreq time.Duration = 100 * time.Millisecond
//)
//
//var (
//	// the term "interactive" is used to mean an item has a dedicated, keyboard-
//	// driven key combo, so that it behaves much like a button.
//	colorScheme = struct {
//		backgroundPrimary   tcell.Color // main background color
//		backgroundSecondary tcell.Color // background color of modal windows
//		backgroundTertiary  tcell.Color // background of dropdown menus, etc.
//		inactiveText        tcell.Color // non-interactive info, secondary or unfocused
//		activeText          tcell.Color // non-interactive info, primary or focused
//		inactiveMenuText    tcell.Color // unselected interactive text
//		activeMenuText      tcell.Color // selected interactive text
//		activeBorder        tcell.Color // border of active/modal views
//		highlightPrimary    tcell.Color // active selections and prominent indicators
//		highlightSecondary  tcell.Color // dynamic persistent status info
//		highlightTertiary   tcell.Color // dynamic temporary status info
//	}{
//		backgroundPrimary:   tcell.ColorBlack,
//		backgroundSecondary: tcell.ColorDarkSlateGray,
//		backgroundTertiary:  tcell.ColorSkyblue,
//		inactiveText:        tcell.ColorDarkSlateGray,
//		activeText:          tcell.ColorWhiteSmoke,
//		inactiveMenuText:    tcell.ColorSkyblue,
//		activeMenuText:      tcell.ColorDodgerBlue,
//		activeBorder:        tcell.ColorSkyblue,
//		highlightPrimary:    tcell.ColorDarkOrange,
//		highlightSecondary:  tcell.ColorDodgerBlue,
//		highlightTertiary:   tcell.ColorGreenYellow,
//	}
//)

func init() {
	// color overrides for the primitives initialized by tview
	tview.Styles.ContrastBackgroundColor = colorScheme.backgroundSecondary
	tview.Styles.MoreContrastBackgroundColor = colorScheme.backgroundTertiary
	tview.Styles.BorderColor = colorScheme.activeText
	tview.Styles.PrimaryTextColor = colorScheme.activeText
}

func busyMessage(intent string) string {
	return fmt.Sprintf("(not ready) please wait until the current operation completes to %s.", intent)
}

// type FocusDelegator defines the methods that must exist for any widget or
// visual entity that can be focused for interaction by the user.
type FocusDelegator interface {
	desc() string // identity for debugging!
	setDelegates(*Layout, FocusDelegator, FocusDelegator)
	page() string
	next() FocusDelegator
	prev() FocusDelegator
	focus() // you must NOT write to the (*Layout).focusQueue from either of
	blur()  // these methods, or you -will- cause deadlock!!
}

// type Layout holds the high level components of the terminal user interface
// as well as the main tview runtime API object tview.Application.
type Layout struct {
	ui     *tview.Application
	option *Options
	lib    []*Library
	busy   *BusyState

	pages     *tview.Pages
	pagesRoot string

	root *tview.Grid

	quitModal  *QuitDialog
	helpInfo   *HelpInfoView
	libSelect  *LibSelectView
	browseView *BrowseView
	logView    *LogView

	focusQueue chan FocusDelegator
	focusLock  sync.Mutex
	focusBase  FocusDelegator
	focused    FocusDelegator

	eventQueue chan func()

	// NOTE: this vars below won't get set until one of the draw routines which
	// uses a tcell.Screen is called, so be careful when accessing them -- make
	// sure they're actually available.
	screen *tcell.Screen
}

// function show() starts drawing the user interface.
func (l *Layout) show() *ReturnCode {

	// zeroized Time is some time in the distant past.
	lastUpdate := time.Time{}

	// queues a screen refresh on the tview event queue and updates the time
	// marker representing the last moment in time we updated.
	redraw := func(f func()) {
		l.ui.QueueUpdateDraw(f)
		lastUpdate = time.Now()
	}

	// timer forcing the app to redraw any areas that may have changed. this
	// update frequency is dynamic -- more frequent while the "Busy" indicator
	// is active, less frequent while it isn't.
	go func(l *Layout) {

		// use the CPU-intensive frequency by default to err on the side of
		// caution.
		updateFreq := busyUpdateFreq

		// updates the currently selected refresh rate only if the requested
		// rate is different from the current.
		setFreq := func(curr, freq *time.Duration) bool {
			if *curr != *freq {
				*curr = *freq
				return true
			}
			return false
		}

		// let's see if i can describe this. lots of nested loops, lots of break
		// statements, not particularly intuitive. most of these loops are
		// infinite and unconditional and serve primarily to poll channels at
		// regular intervals. i'll describe each loop at the level it appears.

		// this outer-most loop simply provides a mechanism to reset the ticker,
		// so that the refresh rate can be changed dynamically:
		//   1. create a ticker with the currently selected frequency
		//   2. start infinite loop, breaking out -only-if- frequency changed.
		//   3. dispose of ticker, return to 1 (we can only reach here if the
		//      frequency changed, breaking out of the outer loop).
		for {
			tick := time.NewTicker(updateFreq)
		REFRESH:
			// this is the actual main draw cycle, iterating only as frequently
			// as our currently selected tick duration -- which should be longer
			// if we are idle, shorter if we are busy and constantly redrawing.
			for {
				select {

				case <-tick.C:
				DRAIN:
					// at each fired tick event, check if we've added any events
					// to our private internal event queue. if so, immediately
					// pull out -all- events, process them, and then perform a
					// single screen draw refresh. this prevents flicker and
					// ensures all updates get handled without starving out all
					// other goroutines.
					for eventsHandled := 0; ; {
						select {
						case event := <-l.eventQueue:
							// we have an event in the queue, so inc the counter
							// to indicate we will need to redraw the screen.
							if event != nil {
								eventsHandled++
								event()
							}
						default:
							// if nothing in the queue, then this for-loop will
							// have its body evaluated only a single time each
							// draw cycle to then break out and check again.

							// we need to redraw the screen because a change was
							// made to one of the primitives.
							shouldDrawForEvent := eventsHandled > 0
							// we need to redraw the screen because
							shouldDrawForTimeout := time.Since(lastUpdate) > idleUpdateFreq
							if shouldDrawForEvent || shouldDrawForTimeout {
								redraw(func() {})
							}
							break DRAIN
						}
					}

				case count := <-l.busy.changed:
					// if the frequency changed, perform one last screen refresh
					// before updating the draw cycle duration. the duration is
					// selected based on the number of goroutines which have
					// incremented the BusyState semaphore. if non-zero, then we
					// have active goroutines and need to frequently redraw the
					// screen. otherwise, we are basically idle and only need to
					// redraw occassionally.
					redraw(func() {})
					// use setFreq() so that we kill the Ticker and alloc a new
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

				// signal monitor for refocus requests. when an event occurs
				// that requires a new widget to be focused, this routine will
				// call the interface-compliant widgets' event handlers to
				// blur() and focus() the old and new widgets, respectively.
				case delegate := <-l.focusQueue:
					if delegate != nil {
						l.focusLock.Lock()
						if delegate != l.focused {
							if l.focused != nil {
								l.focused.blur()
							}
							delegate.focus()
							// update afterwards so that the focus() method can
							// make decisions based on which was previously
							// focused.
							l.focused = delegate
						}
						l.focusLock.Unlock()
						redraw(func() {})
					}
				}
			}
			tick.Stop()
		}
	}(l)

	// the default view to focus when no other view is explicitly requested
	l.focusBase = l.browseView
	l.focusQueue <- l.focusBase

	l.logView.ScrollToEnd()

	if err := l.ui.Run(); err != nil {
		return rcTUIError.specf("show(): ui.Run(): %s", err)
	}
	return nil
}

func stop(ui *tview.Application) {
	if nil != ui {
		ui.Stop()
	}
}

// function newLayout() creates the initial layout of the user interface and
// populates it with the primary widgets. each Library passed in as argument
// has its Layout field initialized with this instance.
func newLayout(opt *Options, busy *BusyState, lib ...*Library) *Layout {

	var layout Layout

	ui := tview.NewApplication()

	header := tview.NewBox().
		SetBorder(false)

	browseView := newBrowseView(ui, "root", lib)
	logView := newLogView(ui, "root", lib)

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
		AddItem(header /******/, 0, 0, 1, 3, 0, 0, false).
		AddItem(browseView /**/, 1, 0, 1, 3, 0, 0, false).
		AddItem(logView /*****/, 2, 0, 1, 3, 0, 0, false).
		AddItem(footer /******/, 3, 0, 1, 3, 0, 0, false)

	root. // other options for the primary layout grid
		SetBorders(true)

	quitModal := newQuitDialog(ui, "quitModal", lib)
	libSelect := newLibSelectView(ui, "libSelect", lib)
	helpInfo := newHelpInfoView(ui, "helpInfo", lib)

	pages := tview.NewPages().
		AddPage("root", root, true, true).
		AddPage(quitModal.page(), quitModal, false, true).
		AddPage(libSelect.page(), libSelect, false, true).
		AddPage(helpInfo.page(), helpInfo, false, true)

	header. // register the header bar screen drawing callback
		SetDrawFunc(layout.drawMenuBar)

	footer. // register the status bar screen drawing callback
		SetDrawFunc(layout.drawStatusBar)

	// define the higher-order tab cycle
	browseView.setDelegates(&layout, nil, nil)
	logView.setDelegates(&layout, nil, nil)
	quitModal.setDelegates(&layout, nil, nil)
	libSelect.setDelegates(&layout, nil, nil)
	helpInfo.setDelegates(&layout, nil, nil)

	// and finally initialize our actual Layout object to be returned
	layout = Layout{
		ui:     ui,
		option: opt,
		lib:    lib,
		busy:   busy,

		pages:     pages,
		pagesRoot: "root",

		root: root,

		quitModal:  quitModal,
		helpInfo:   helpInfo,
		libSelect:  libSelect,
		browseView: browseView,
		logView:    logView,

		focusQueue: make(chan FocusDelegator),
		focusLock:  sync.Mutex{},
		focusBase:  nil,
		focused:    nil,

		eventQueue: make(chan func()),

		screen: nil,
	}

	// add a ref to this layout object to all libraries
	//for _, l := range lib {
	//	l.layout = &layout
	//}

	// set the initial page displayed when application begins
	pages.SwitchToPage(layout.pagesRoot)

	ui. // global tview application configuration
		SetRoot(pages, true).
		SetInputCapture(layout.inputEvent)

	// manually initiate the event handler for selecting the "(All)"-libraries
	// dropdown to update the meta info in the LibSelectView
	libSelect.
		selectedLibDropDown(selectedLibraryAllOption, selectedLibraryAll)

	return &layout
}

func (l *Layout) shouldDelegateInputEvent(busy bool, event *tcell.EventKey) bool {

	l.focusLock.Lock()
	focused := l.focused
	l.focusLock.Unlock()

	switch focused.(type) {
	case *LogView:
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h', 'H', 'j', 'J', 'k', 'K', 'l', 'L':
				// do NOT support the vi-style navigation keys in the log view
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

	focusWidget := map[rune]FocusDelegator{
		'L': l.libSelect,
		'H': l.helpInfo,
		'V': l.logView,
	}

	fwdEvent := event
	isBusy := l.busy.count() > 0

	l.focusLock.Lock()
	focused := l.focused
	l.focusLock.Unlock()

	evTime := event.When()
	evMod := event.Modifiers()
	evKey := event.Key()
	evRune := event.Rune()

	// catch some global, application-level events before evaluating them in the
	// context of whatever view is currently focused.
	switch {
	case tcell.KeyCtrlC == evKey:
		// don't exit on Ctrl+C, it feels unsanitary. instead, notify the
		// user we can exit cleanly by simply pressing 'q'.
		fwdEvent = nil
		warnLog.logf("(ignored) please use '%c' key to terminate the "+
			"application. ctrl keys are swallowed to prevent choking.", 'q')
	}

	navigationEvent := func(lo *Layout, busy bool, ek tcell.Key, er rune, em tcell.ModMask, et time.Time) bool {
		switch ek {
		case tcell.KeyRune:
			if widget, ok := focusWidget[unicode.ToUpper(er)]; ok {
				// do not process any navigation events (opening windows, dialogs, etc.)
				// if our BusyState indicates we are preoccupied handling other events,
				// unless the view we are wanting to access is the HelpView.
				if busy && (widget != l.helpInfo) {
					warnLog.logf(busyMessage("navigate or open a submenu"))
					return false
				}
				lo.focusQueue <- widget
				return true
			}
			// TODO: remove me, exists only for eval of color palettes
			if fn, ok := logColors[er]; ok {
				fn()
				return true
			}
		}
		return false
	}

	exitEvent := func(lo *Layout, ek tcell.Key, er rune, em tcell.ModMask, et time.Time) bool {
		switch ek {
		case tcell.KeyRune:
			switch er {
			case 'q', 'Q':
				return true
			}
		}
		return false
	}

	switch focused.(type) {

	case *HelpInfoView:
		if !navigationEvent(l, isBusy, evKey, evRune, evMod, evTime) {
			switch evKey {
			case tcell.KeyEsc, tcell.KeyRune:
				l.focusQueue <- l.focusBase
			}
			if exitEvent(l, evKey, evRune, evMod, evTime) {
				l.focusQueue <- l.quitModal
			}
		}

	case *LibSelectView:
		switch evKey {
		case tcell.KeyEsc:
			l.focusQueue <- l.focusBase
		}

	case *BrowseView:
		if !navigationEvent(l, isBusy, evKey, evRune, evMod, evTime) {
			switch evKey {
			case tcell.KeyEsc:
				l.focusQueue <- l.focusBase
			}
			if exitEvent(l, evKey, evRune, evMod, evTime) {
				l.focusQueue <- l.quitModal
			}
		}

	case *LogView:
		if !navigationEvent(l, isBusy, evKey, evRune, evMod, evTime) {
			switch evKey {
			case tcell.KeyEsc:
				l.focusQueue <- l.focusBase
			}
			if exitEvent(l, evKey, evRune, evMod, evTime) {
				l.focusQueue <- l.quitModal
			}
		}
	}

	if !l.shouldDelegateInputEvent(isBusy, event) {
		fwdEvent = nil
	}

	return fwdEvent
}

// function drawMenuBar() is the callback handler associated with the top-most
// header box. this routine is not called on-demand, but is usually invoked
// implicitly by other re-draw events.
func (l *Layout) drawMenuBar(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	const (
		libDimWidth   = 40 // library selection window width
		libDimHeight  = 20 // ^----------------------- height
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

	libName := l.libSelect.selectedName
	library := fmt.Sprintf("[::bu]%s[::-]%s: [#%06x]%s", "L", "ibrary", colorScheme.highlightPrimary.Hex(), libName)
	help := fmt.Sprintf("[::bu]%s[::-]%s", "H", "elp")

	tview.Print(screen, library, x+3, y, width, tview.AlignLeft, colorScheme.inactiveMenuText)
	tview.Print(screen, help, x, y, width-3, tview.AlignRight, colorScheme.inactiveMenuText)

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

// function drawStatusBar() is the callback handler associated with the bottom-
// most footer box. this routine is regularly called so that the datetime clock
// remains accurate along with any status information currently available.
// this function is the primary driver of the BusyState's UI cycle counter.
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
	dateTime := time.Now().Format("2006/01/02 03:04 PM")

	// Write some text along the horizontal line.
	tview.Print(screen, dateTime, x+3, y, width, tview.AlignLeft, colorScheme.highlightSecondary)

	// update the busy indicator if we have any active worker threads
	count := l.busy.count()
	if count > 0 {
		// increment the screen refresh counter
		cycle := l.busy.next()

		// draw the "working..." indicator. note the +2 is to make room for the
		// moon rune following this indicator.
		working := fmt.Sprintf("working%-*s", ellipses, bytes.Repeat([]byte{'.'}, cycle%ellipses))
		tview.Print(screen, working, x-ellipses+1, y, width, tview.AlignRight, colorScheme.highlightTertiary)

		// draw the cyclic moon rotation
		moon := fmt.Sprintf("%c ", MoonPhase[cycle%MoonPhaseLength])
		tview.Print(screen, moon, x, y, width, tview.AlignRight, colorScheme.highlightPrimary)
	}

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

func (l *Layout) addDiscovery(lib *Library, disco *Discovery) *ReturnCode {

	var media *Media = nil

	switch disco.data[0].(type) {
	case *AudioMedia:
		audio := disco.data[0].(*AudioMedia)
		media = audio.Media
	case *VideoMedia:
		video := disco.data[0].(*VideoMedia)
		media = video.Media
	case *Subtitles:
		_ = disco.data[0].(*Subtitles) // TBD: unused currently
	}

	if nil != media {
		l.eventQueue <- func() {
			position, primary, secondary := l.browseView.positionForMediaItem(media)
			l.browseView.insertMediaItem(lib, media, position, primary, secondary, nil)
		}
	}

	return nil
}

//------------------------------------------------------------------------------

type QuitDialog struct {
	*tview.Modal
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newQuitDialog() allocates and initializes the tview.Modal widget
// that prompts the user to confirm before quitting the application.
func newQuitDialog(ui *tview.Application, page string, lib []*Library) *QuitDialog {

	prompt := "Oh, so you're a quitter, huh?"
	button := []string{"Y-yeah...", " Fuck NO "}

	view := tview.NewModal().
		SetText(prompt).
		AddButtons(button)

	v := QuitDialog{view, nil, page, nil, nil}

	v.SetDoneFunc(
		func(buttonIndex int, buttonLabel string) {
			switch {
			case button[0] == buttonLabel:
				stop(ui)
			case button[1] == buttonLabel:
				v.layout.focusQueue <- v.layout.focusBase
			}
		})

	return &v
}

func (v *QuitDialog) desc() string { return "" }
func (v *QuitDialog) setDelegates(layout *Layout, prev, next FocusDelegator) {
	v.layout = layout
	v.focusPrev = prev
	v.focusNext = next
}
func (v *QuitDialog) page() string         { return v.focusPage }
func (v *QuitDialog) next() FocusDelegator { return v.focusNext }
func (v *QuitDialog) prev() FocusDelegator { return v.focusPrev }
func (v *QuitDialog) focus() {
	page := v.page()
	v.layout.pages.ShowPage(page)
}
func (v *QuitDialog) blur() {
	page := v.page()
	v.layout.pages.HidePage(page)
}

//------------------------------------------------------------------------------

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
func newHelpInfoView(ui *tview.Application, page string, lib []*Library) *HelpInfoView {

	v := HelpInfoView{nil, nil, page, nil, nil}

	help := tview.NewBox()

	help.
		SetBorder(true).
		SetBorderColor(colorScheme.activeBorder).
		SetTitle(" Help ").
		SetTitleColor(colorScheme.activeMenuText).
		SetTitleAlign(tview.AlignRight)

	help.
		SetDrawFunc(v.drawHelpInfoView)

	v.Box = help

	return &v
}

func (v *HelpInfoView) desc() string { return "" }
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

	tview.Print(screen, swvers, x+1, y, width-2, tview.AlignLeft, colorScheme.highlightPrimary)

	// Coordinate space for subsequent draws.
	return 0, 0, 0, 0
}

//------------------------------------------------------------------------------

type LibSelectViewFormItem int

const (
	lsiLibrary LibSelectViewFormItem = iota
	lsiCOUNT
)

// the index used to indicate -all- libraries selected for viewing.
const selectedLibraryAll = 0
const selectedLibraryAllOption = "(All)"

type LibSelectView struct {
	*tview.Form
	libDropDown *tview.DropDown
	layout      *Layout
	focusPage   string
	focusNext   FocusDelegator
	focusPrev   FocusDelegator

	library         []*Library
	selectedLibrary int
	selectedName    string
	numTotal        uint
	numVideo        uint
	numAudio        uint
}

// function makeUniqueLibraryNames() creates unambiguous library names for all
// given libraries. this is achieved by starting with the right-most component
// of each path and iteratively adding its parent directory until the paths
// can be uniquely identified.
func makeUniqueLibraryNames(library []*Library) []string {

	reverse := func(a []string) []string {
		for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
			a[i], a[j] = a[j], a[i]
		}
		return a
	}

	type indexedSlice struct {
		index  int
		slice  []string
		joined string
	}

	name := make([]indexedSlice, len(library))
	maxLength := 0
	for i := range name {
		component := strings.Split(strings.TrimRight(library[i].absPath, pathSep), pathSep)
		name[i] = indexedSlice{1, reverse(component), ""}
		if length := len(component); length > maxLength {
			maxLength = length
		}
	}

	for n := 0; n < maxLength; n++ {
		d := map[string]int{}
		for i := range name {

			index := name[i].index
			slice := name[i].slice[:index]
			name[i].joined = strings.Join(slice, pathSep)
			d[name[i].joined]++
		}
		for i := range name {
			if d[name[i].joined] > 1 {
				name[i].index++
			}
		}
		if len(d) == len(name) {
			result := []string{}
			for _, n := range name {
				joined := reverse(strings.Split(n.joined, pathSep))
				result = append(result, strings.Join(joined, pathSep))
			}
			return result
		}
	}

	// there was an error in the algorithm if we reached here.
	return []string{}
}

// function newLibSelectView() allocates and initializes the tview.Form widget
// where the user selects which library to browse and any other filtering
// options.
func newLibSelectView(ui *tview.Application, page string, lib []*Library) *LibSelectView {

	unique := makeUniqueLibraryNames(lib)
	libName := []string{selectedLibraryAllOption}
	dropDownWidth := len(selectedLibraryAllOption)
	for _, u := range unique {
		if n := len(u); n > dropDownWidth {
			dropDownWidth = n
		}
		libName = append(libName, u)
	}

	for i, name := range libName { // +3 strictly for formatting/appearance
		libName[i] = fmt.Sprintf("%-*s", dropDownWidth+3, name)
	}

	// offset library by 1 so that the "all" item is at index 0.
	xref := make([]*Library, len(lib)+1)
	copy(xref[1:], lib)

	v :=
		LibSelectView{
			Form:            nil,
			libDropDown:     nil,
			layout:          nil,
			focusPage:       page,
			focusNext:       nil,
			focusPrev:       nil,
			library:         xref,
			selectedLibrary: selectedLibraryAll,
			selectedName:    selectedLibraryAllOption,
			numTotal:        0,
			numVideo:        0,
			numAudio:        0,
		}

	form := tview.NewForm().
		AddDropDown("   Show:", libName, 0, v.selectedLibDropDown).
		SetLabelColor(colorScheme.inactiveMenuText).
		SetFieldTextColor(colorScheme.inactiveMenuText).
		SetFieldBackgroundColor(colorScheme.backgroundSecondary)

	form.
		SetBorder(true).
		SetBorderColor(colorScheme.activeBorder).
		SetTitleColor(colorScheme.activeMenuText).
		SetTitleAlign(tview.AlignLeft).
		SetDrawFunc(v.drawLibSelectView)

	v.Form = form
	v.libDropDown = form.GetFormItem(int(lsiLibrary)).(*tview.DropDown)

	for i := 0; i < int(lsiCOUNT); i++ {
		f := v.GetFormItem(i)
		if nil == f {
			break
		}
		switch f.(type) {
		case *tview.InputField:
			f.(*tview.InputField).SetInputCapture(v.inputFieldInput)
		case *tview.DropDown:
			f.(*tview.DropDown).SetInputCapture(v.dropDownInput)
		}
	}

	return &v
}

func (v *LibSelectView) desc() string { return "" }
func (v *LibSelectView) setDelegates(layout *Layout, prev, next FocusDelegator) {
	v.layout = layout
	v.focusPrev = prev
	v.focusNext = next
}
func (v *LibSelectView) page() string         { return v.focusPage }
func (v *LibSelectView) next() FocusDelegator { return v.focusNext }
func (v *LibSelectView) prev() FocusDelegator { return v.focusPrev }
func (v *LibSelectView) focus() {
	// first update the library media counters upon focus of this view.
	switch selected := v.library[v.selectedLibrary]; v.selectedLibrary {
	case selectedLibraryAll:
		v.updateMediaCount(v.library...)
	default:
		if nil != selected {
			v.updateMediaCount(selected)
		}
	}
	page := v.page()
	v.layout.pages.ShowPage(page)
}
func (v *LibSelectView) blur() {
	page := v.page()
	v.layout.pages.HidePage(page)
}

// function updateMediaCount() iterates over the given libraries and counts the
// number of each kind of media discovered. the LibSelectView object's counts
// are immediately updated for reading.
func (v *LibSelectView) updateMediaCount(library ...*Library) {

	v.numVideo = 0
	v.numAudio = 0

	for _, l := range library {
		if nil != l {
			v.numVideo +=
				l.db.numRecordsLoad[ecMedia][mkVideo] +
					l.db.numRecordsScan[ecMedia][mkVideo]

			v.numAudio +=
				l.db.numRecordsLoad[ecMedia][mkAudio] +
					l.db.numRecordsScan[ecMedia][mkAudio]
		}
	}

	v.numTotal = v.numVideo + v.numAudio
}
func (v *LibSelectView) drawLibSelectView(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	const (
		libDimWidth   = 40 // library selection window width
		libDimHeight  = 20 // ^----------------------- height
		helpDimWidth  = 40 // help info window width
		helpDimHeight = 10 // ^--------------- height
	)

	// update the layout's associated screen field. note that you must be very
	// careful and not access this field until this status line has been drawn
	// at least one time.
	if nil == v.layout.screen {
		v.layout.screen = &screen
	}

	v.SetTitle(fmt.Sprintf(" Library: [#%06x]%s ", colorScheme.highlightPrimary.Hex(), v.selectedName))

	// any existing library scan times must have occurred before right now.
	lastScan := time.Now()
	selectedLibrary := v.library[v.selectedLibrary]
	if nil != selectedLibrary {
		lastScan = selectedLibrary.lastScan
	} else {
		// if showing all libraries, display the -oldest- scan time as it is the
		// most conservative choice.
		for _, l := range v.library {
			if nil != l {
				if l.lastScan.Before(lastScan) {
					lastScan = l.lastScan
				}
			}
		}
	}

	ddX, ddY, _, _ := v.libDropDown.GetRect()

	fmtInfoRow := func(label, value string) string {
		return fmt.Sprintf("[#%06x]%10s: [#%06x]%s",
			colorScheme.inactiveMenuText.Hex(), label,
			colorScheme.highlightPrimary.Hex(), value)
	}

	for i, s := range []string{
		fmtInfoRow("Video", strconv.FormatUint(uint64(v.numVideo), 10)),
		fmtInfoRow("Audio", strconv.FormatUint(uint64(v.numAudio), 10)),
		fmtInfoRow("Last scan", lastScan.Format("2006/01/02 15:04:05")),
	} {
		tview.Print(screen, s, ddX+3, ddY+2+i, width, tview.AlignLeft, colorScheme.inactiveMenuText)
	}

	return v.Form.GetInnerRect()
}
func (v *LibSelectView) selectedLibDropDown(option string, optionIndex int) {

	// do not handle any dropdown selection if we are preoccupied handling some
	// other event or request.
	if isBusy := v.layout.busy.count() > 0; isBusy {
		return
	}

	// immediately and unconditionally set the class-visible variable which
	// holds the user-selected library index.
	v.selectedLibrary = optionIndex

	// include all libraries by default, and then filter the list down based on
	// user selections.
	includedLib := v.library

	// if optionIndex is selectedLibraryAll(0), then the value of "selected"
	// should be nil, which showLibrary() interprets as "all libraries".
	selected := includedLib[optionIndex]
	if optionIndex != selectedLibraryAll {
		if nil == selected {
			// this is an error; we should always have a Library at the selected
			// option index unless it is index 0 (i.e. selectedLibraryAll).
			return
		}
		// user selected an option that -isn't- the "(All)"-libraries selection,
		// so only show that one selected Library.
		includedLib = []*Library{selected}
	}

	// update the selected library data and content counters based on the list
	// of selected libraries.
	v.updateMediaCount(includedLib...)
	v.selectedName = strings.TrimSpace(option)
	go func() {
		// protect the libraries from being modified while we are updating the
		// media browser and library selection.
		v.layout.busy.inc()
		v.layout.browseView.showLibrary(selected)
		v.layout.busy.dec()
	}()
}
func (v *LibSelectView) inputFieldInput(event *tcell.EventKey) *tcell.EventKey {
	isBusy := v.layout.busy.count() > 0
	switch key := event.Key(); key {
	case tcell.KeyDown:
		// treat the down arrow as a tab key for simpler navigation through the
		// form items.
		return tcell.NewEventKey(tcell.KeyTab, 0, event.Modifiers())
	case tcell.KeyUp:
		// treat the up arrow as a backtab key for simpler navigation through
		// the form items.
		return tcell.NewEventKey(tcell.KeyBacktab, 0, event.Modifiers())
	default:
		// do not allow the user to edit any input fields until we have finished
		// processing whatever has flagged out BusyState indicator.
		if isBusy {
			event = nil
		}
	}
	return event
}
func (v *LibSelectView) dropDownInput(event *tcell.EventKey) *tcell.EventKey {
	isBusy := v.layout.busy.count() > 0
	switch key := event.Key(); key {
	case tcell.KeyRune:
		// just ignore any character keys pressed, do not perform the default
		// (annoying) prefix-processing of the DropDown.
		event = nil
	case tcell.KeyEnter:
		// do not allow the user to select a new library until we have finished
		// processing whatever has flagged our BusyState indicator.
		if isBusy {
			warnLog.logf(busyMessage("select a new library"))
			event = nil
		}
	case tcell.KeyDown, tcell.KeyUp:
		// handle the up/down keys with the same event handler as the input
		// fields so that they behave like tab/backtab. the user must press the
		// Enter key to actually access the DropDown items.
		return v.inputFieldInput(tcell.NewEventKey(key, 0, event.Modifiers()))
	}
	return event
}

//------------------------------------------------------------------------------

type BrowseView struct {
	*Browser
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newBrowseView() allocates and initializes the tview.List widget
// where all of the currently available media can be browsed.
func newBrowseView(ui *tview.Application, page string, lib []*Library) *BrowseView {

	list := newBrowser()
	v := BrowseView{list, nil, page, nil, nil}
	v.setSelectedFunc(v.selectItem)

	return &v
}

func (v *BrowseView) desc() string { return "" }
func (v *BrowseView) setDelegates(layout *Layout, prev, next FocusDelegator) {
	v.layout = layout
	v.focusPrev = prev
	v.focusNext = next
}
func (v *BrowseView) page() string         { return v.focusPage }
func (v *BrowseView) next() FocusDelegator { return v.focusNext }
func (v *BrowseView) prev() FocusDelegator { return v.focusPrev }
func (v *BrowseView) focus() {
	page := v.page()
	v.layout.pages.ShowPage(page)
	v.layout.ui.SetFocus(v.Browser)
}
func (v *BrowseView) blur()                                                {}
func (v *BrowseView) selectItem(index int, mainText, secondaryText string) {}

//------------------------------------------------------------------------------

type LogView struct {
	*tview.TextView
	layout    *Layout
	focusPage string
	focusNext FocusDelegator
	focusPrev FocusDelegator
}

// function newLogView() allocates and initializes the tview.TextView widget
// where all runtime log data is navigated by and displayed to the user.
func newLogView(ui *tview.Application, page string, lib []*Library) *LogView {

	logChanged := func() {}
	logDone := func(key tcell.Key) {}

	view := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(colorScheme.inactiveText).
		SetWordWrap(true).
		SetWrap(false)

	view. // update the TextView event handlers
		SetChangedFunc(logChanged).
		SetDoneFunc(logDone).
		SetBorder(false)

	v := LogView{view, nil, page, nil, nil}

	return &v
}

func (v *LogView) desc() string { return "" }
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
	v.TextView.SetTextColor(colorScheme.activeText)
}
func (v *LogView) blur() {
	v.TextView.SetTextColor(colorScheme.inactiveText)
}

// -----------------------------------------------------------------------------
//  TBD: temporary code below while evaluating color palettes
// -----------------------------------------------------------------------------

var logColors = map[rune]func(){
	'1': func() {
		// blue|navy
		infoLog.log("[#000080]ColorNavy")
		infoLog.log("[#00008b]ColorDarkBlue")
		infoLog.log("[#0000cd]ColorMediumBlue")
		infoLog.log("[#0000ff]ColorBlue")
		infoLog.log("[#00bfff]ColorDeepSkyBlue")
		infoLog.log("[#191970]ColorMidnightBlue")
		infoLog.log("[#1e90ff]ColorDodgerBlue")
		infoLog.log("[#4169e1]ColorRoyalBlue")
		infoLog.log("[#4682b4]ColorSteelBlue")
		infoLog.log("[#483d8b]ColorDarkSlateBlue")
		infoLog.log("[#5f9ea0]ColorCadetBlue")
		infoLog.log("[#6495ed]ColorCornflowerBlue")
		infoLog.log("[#6a5acd]ColorSlateBlue")
		infoLog.log("[#7b68ee]ColorMediumSlateBlue")
		infoLog.log("[#87ceeb]ColorSkyblue")
		infoLog.log("[#87cefa]ColorLightSkyBlue")
		infoLog.log("[#8a2be2]ColorBlueViolet")
		infoLog.log("[#add8e6]ColorLightBlue")
		infoLog.log("[#b0c4de]ColorLightSteelBlue")
		infoLog.log("[#b0e0e6]ColorPowderBlue")
		infoLog.log("[#f0f8ff]ColorAliceBlue")
	},

	'2': func() {
		// red|pink|magenta|fire|crimson|tomato|salmon|coral|maroon|rose|seashell
		infoLog.log("[#800000]ColorMaroon")
		infoLog.log("[#8b0000]ColorDarkRed")
		infoLog.log("[#8b008b]ColorDarkMagenta")
		infoLog.log("[#b22222]ColorFireBrick")
		infoLog.log("[#c71585]ColorMediumVioletRed")
		infoLog.log("[#cd5c5c]ColorIndianRed")
		infoLog.log("[#db7093]ColorPaleVioletRed")
		infoLog.log("[#dc143c]ColorCrimson")
		infoLog.log("[#e9967a]ColorDarkSalmon")
		infoLog.log("[#f08080]ColorLightCoral")
		infoLog.log("[#fa8072]ColorSalmon")
		infoLog.log("[#ff0000]ColorRed")
		infoLog.log("[#ff1493]ColorDeepPink")
		infoLog.log("[#ff4500]ColorOrangeRed")
		infoLog.log("[#ff6347]ColorTomato")
		infoLog.log("[#ff69b4]ColorHotPink")
		infoLog.log("[#ff7f50]ColorCoral")
		infoLog.log("[#ffa07a]ColorLightSalmon")
		infoLog.log("[#ffb6c1]ColorLightPink")
		infoLog.log("[#ffc0cb]ColorPink")
		infoLog.log("[#ffe4e1]ColorMistyRose")
		infoLog.log("[#fff5ee]ColorSeashell")
	},

	'3': func() {
		// black|white|gray|grey|smoke|silver|gainsboro|linen|oldlace|snow|ivory
		infoLog.log("[#000000]ColorBlack")
		infoLog.log("[#2f4f4f]ColorDarkSlateGray")
		infoLog.log("[#696969]ColorDimGray")
		infoLog.log("[#708090]ColorSlateGray")
		infoLog.log("[#778899]ColorLightSlateGray")
		infoLog.log("[#808080]ColorGray")
		infoLog.log("[#a9a9a9]ColorDarkGray")
		infoLog.log("[#c0c0c0]ColorSilver")
		infoLog.log("[#d3d3d3]ColorLightGray")
		infoLog.log("[#dcdcdc]ColorGainsboro")
		infoLog.log("[#f5f5f5]ColorWhiteSmoke")
		infoLog.log("[#f8f8ff]ColorGhostWhite")
		infoLog.log("[#faebd7]ColorAntiqueWhite")
		infoLog.log("[#faf0e6]ColorLinen")
		infoLog.log("[#fdf5e6]ColorOldLace")
		infoLog.log("[#ffdead]ColorNavajoWhite")
		infoLog.log("[#fffaf0]ColorFloralWhite")
		infoLog.log("[#fffafa]ColorSnow")
		infoLog.log("[#fffff0]ColorIvory")
		infoLog.log("[#ffffff]ColorWhite")
	},

	'4': func() {
		// green|lime|olive|chartreuse|mint
		infoLog.log("[#006400]ColorDarkGreen")
		infoLog.log("[#008000]ColorGreen")
		infoLog.log("[#00fa9a]ColorMediumSpringGreen")
		infoLog.log("[#00ff00]ColorLime")
		infoLog.log("[#00ff7f]ColorSpringGreen")
		infoLog.log("[#20b2aa]ColorLightSeaGreen")
		infoLog.log("[#228b22]ColorForestGreen")
		infoLog.log("[#2e8b57]ColorSeaGreen")
		infoLog.log("[#32cd32]ColorLimeGreen")
		infoLog.log("[#3cb371]ColorMediumSeaGreen")
		infoLog.log("[#556b2f]ColorDarkOliveGreen")
		infoLog.log("[#6b8e23]ColorOliveDrab")
		infoLog.log("[#7cfc00]ColorLawnGreen")
		infoLog.log("[#7fff00]ColorChartreuse")
		infoLog.log("[#808000]ColorOlive")
		infoLog.log("[#8fbc8f]ColorDarkSeaGreen")
		infoLog.log("[#90ee90]ColorLightGreen")
		infoLog.log("[#98fb98]ColorPaleGreen")
		infoLog.log("[#9acd32]ColorYellowGreen")
		infoLog.log("[#adff2f]ColorGreenYellow")
		infoLog.log("[#f5fffa]ColorMintCream")
	},

	'5': func() {
		// turquoise|teal|cyan|aqua|azure
		infoLog.log("[#008080]ColorTeal")
		infoLog.log("[#008b8b]ColorDarkCyan")
		infoLog.log("[#00ced1]ColorDarkTurquoise")
		infoLog.log("[#00ffff]ColorAqua")
		infoLog.log("[#40e0d0]ColorTurquoise")
		infoLog.log("[#48d1cc]ColorMediumTurquoise")
		infoLog.log("[#66cdaa]ColorMediumAquamarine")
		infoLog.log("[#7fffd4]ColorAquaMarine")
		infoLog.log("[#afeeee]ColorPaleTurquoise")
		infoLog.log("[#e0ffff]ColorLightCyan")
		infoLog.log("[#f0ffff]ColorAzure")
	},

	'6': func() {
		// purple|indigo|violet|lavender|fuchsia|orchid|thistle|plum
		infoLog.log("[#4b0082]ColorIndigo")
		infoLog.log("[#663399]ColorRebeccaPurple")
		infoLog.log("[#800080]ColorPurple")
		infoLog.log("[#9370db]ColorMediumPurple")
		infoLog.log("[#9400d3]ColorDarkViolet")
		infoLog.log("[#9932cc]ColorDarkOrchid")
		infoLog.log("[#ba55d3]ColorMediumOrchid")
		infoLog.log("[#d8bfd8]ColorThistle")
		infoLog.log("[#da70d6]ColorOrchid")
		infoLog.log("[#dda0dd]ColorPlum")
		infoLog.log("[#e6e6fa]ColorLavender")
		infoLog.log("[#ee82ee]ColorViolet")
		infoLog.log("[#ff00ff]ColorFuchsia")
		infoLog.log("[#fff0f5]ColorLavenderBlush")
	},

	'7': func() {
		// yellow|gold|corn|lemon|papaya|orange|peach|honeydew
		infoLog.log("[#b8860b]ColorDarkGoldenrod")
		infoLog.log("[#daa520]ColorGoldenrod")
		infoLog.log("[#eee8aa]ColorPaleGoldenrod")
		infoLog.log("[#f0fff0]ColorHoneydew")
		infoLog.log("[#fafad2]ColorLightGoldenrodYellow")
		infoLog.log("[#ff8c00]ColorDarkOrange")
		infoLog.log("[#ffa500]ColorOrange")
		infoLog.log("[#ffd700]ColorGold")
		infoLog.log("[#ffdab9]ColorPeachPuff")
		infoLog.log("[#ffefd5]ColorPapayaWhip")
		infoLog.log("[#fff8dc]ColorCornsilk")
		infoLog.log("[#fffacd]ColorLemonChiffon")
		infoLog.log("[#ffff00]ColorYellow")
		infoLog.log("[#ffffe0]ColorLightYellow")
	},

	'8': func() {
		// brown|wheat|tan|sienna|peru|moccasin|bisque
		infoLog.log("[#8b4513]ColorSaddleBrown")
		infoLog.log("[#a0522d]ColorSienna")
		infoLog.log("[#a52a2a]ColorBrown")
		infoLog.log("[#bc8f8f]ColorRosyBrown")
		infoLog.log("[#bdb76b]ColorDarkKhaki")
		infoLog.log("[#cd853f]ColorPeru")
		infoLog.log("[#d2691e]ColorChocolate")
		infoLog.log("[#d2b48c]ColorTan")
		infoLog.log("[#deb887]ColorBurlyWood")
		infoLog.log("[#f0e68c]ColorKhaki")
		infoLog.log("[#f4a460]ColorSandyBrown")
		infoLog.log("[#f5deb3]ColorWheat")
		infoLog.log("[#f5f5dc]ColorBeige")
		infoLog.log("[#ffe4b5]ColorMoccasin")
		infoLog.log("[#ffe4c4]ColorBisque")
		infoLog.log("[#ffebcd]ColorBlanchedAlmond")
	},
}
