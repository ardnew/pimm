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

	"fmt"
	"time"
)

// type Layout holds the high level components of the terminal user interface
// as well as the main tview runtime API object tview.Application.
type Layout struct {
	ui   *tview.Application
	grid *tview.Grid
	log  *UILog

	busy *BusyState

	// NOTE: this screen var won't get set until one of the draw routines which
	// uses it is called, so be careful when accessing it -- make sure it's
	// actually available.
	screen *tcell.Screen
}

// function show() starts drawing the user interface.
func (l *Layout) show() *ReturnCode {

	// timer forcing the app to redraw (@ 10Hz) any areas that may have changed.
	go func(l *Layout) {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				l.ui.QueueUpdateDraw(func() {})
			}
		}
	}(l)

	if err := l.ui.Run(); err != nil {
		return rcTUIError.specf("show(): ui.Run(): %s", err)
	}
	return nil
}

// function newLayout() creates the initial layout of the user interface and
// populates it with the primary widgets. each Library passed in as argument
// has its Layout field initialized with this instance.
func newLayout(lib ...*Library) *Layout {

	ui := tview.NewApplication()

	header := tview.NewBox()

	menu := tview.NewBox()
	main := tview.NewBox()
	sideBar := tview.NewBox()

	log := newUILog(ui)
	footer := tview.NewBox()

	grid := tview.NewGrid().
		// these are actual sizes, in terms of addressable terminal locations,
		// i.e. characters and lines. the literal width and height values in the
		// arguments to AddItem() are the logical sizes, in terms of rows and
		// columns that are laid out by the arguments to SetRows()/SetColumns().
		SetRows(3, 0, 12, 1).
		SetColumns(32, 0, 32).
		SetBorders(true).
		// fixed components that are always visible
		AddItem(header /***/, 0, 0, 1, 3, 0, 0, false).
		AddItem(log /******/, 2, 0, 1, 3, 0, 0, false).
		AddItem(footer /***/, 3, 0, 1, 3, 0, 0, false)

	// layout for screens narrower than 100 cells.
	grid.AddItem(menu /*****/, 0, 0, 0, 0, 0, 0, false).
		AddItem(main /******/, 1, 0, 1, 3, 0, 0, false).
		AddItem(sideBar /***/, 0, 0, 0, 0, 0, 0, false)

	// layout for screens wider than 100 cells.
	grid.AddItem(menu /*****/, 1, 0, 1, 1, 0, 100, false).
		AddItem(main /******/, 1, 1, 1, 1, 0, 100, false).
		AddItem(sideBar /***/, 1, 2, 1, 1, 0, 100, false)

	ui.SetRoot(grid, true)
	ui.SetFocus(log)

	layout := Layout{
		ui:     ui,
		grid:   grid,
		log:    log,
		busy:   nil,
		screen: nil,
	}

	footer.SetDrawFunc(layout.drawStatus)

	for _, l := range lib {
		l.layout = &layout
		layout.busy = l.busyState
	}

	return &layout
}

// function drawStatus() is the callback handler associated with the bottom-most
// footer box. this routine is regularly called so that the datetime clock
// remains accurate along with any status information currently available.
func (l *Layout) drawStatus(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

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
		index := l.busy.next() % WaitCycleLength
		waitRune := fmt.Sprintf(" %c ", WaitCycle[index])
		tview.Print(screen, waitRune, x, y, width, tview.AlignLeft, tcell.ColorYellow)
	}

	// Space for other content.
	return 0, 0, 0, 0
}

// type UIPrimitive defines the related fields describing any visual widget or
// component displayed in a Layout.
type UIPrimitive struct {
	box *tview.Box
}

// function newUIPrimitive() allocates and initializes all default data for all
// widgets displayed to the user.
func newUIPrimitive() *UIPrimitive {
	return &UIPrimitive{
		box: tview.NewBox(),
	}
}

// type UILog defines the tview.TextView widget where all runtime log data is
// navigated by and displayed to the user.
type UILog struct {
	*UIPrimitive
	*tview.TextView
}

// function newUILog() allocates and initializes all default data for a runtime
// log view.
func newUILog(ui *tview.Application) *UILog {

	tvChanged := func() { /*ui.Draw()*/ }
	tvDone := func(key tcell.Key) {}

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGhostWhite).
		SetWordWrap(true).
		SetWrap(false)

	tv.SetChangedFunc(tvChanged)
	tv.SetDoneFunc(tvDone)

	return &UILog{
		UIPrimitive: newUIPrimitive(),
		TextView:    tv,
	}
}
