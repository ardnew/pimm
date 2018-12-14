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
	"time"
)

// type Layout holds the high level components of the terminal user interface
// as well as the main tview runtime API object tview.Application.
type Layout struct {
	ui      *tview.Application
	grid    *tview.Grid
	options *Options
	busy    *BusyState
	log     *UILog

	// NOTE: this screen var won't get set until one of the draw routines which
	// uses it is called, so be careful when accessing it -- make sure it's
	// actually available.
	screen *tcell.Screen
}

var (
	idleUpdateFreq time.Duration = 1 * time.Second
	busyUpdateFreq time.Duration = 100 * time.Millisecond
)

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
		ui:      ui,
		grid:    grid,
		options: opt,
		busy:    busy,
		log:     log,
		screen:  nil,
	}

	footer.SetDrawFunc(layout.drawStatus)

	for _, l := range lib {
		l.layout = &layout
	}

	return &layout
}

// function drawStatus() is the callback handler associated with the bottom-most
// footer box. this routine is regularly called so that the datetime clock
// remains accurate along with any status information currently available.
func (l *Layout) drawStatus(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	// the number of ellipsis periods (+1) to draw after the "working" indicator
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
