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
)

// type Layout holds the high level components of the terminal user interface
// as well as the main tview runtime API object tview.Application.
type Layout struct {
	ui   *tview.Application
	grid *tview.Grid
	log  *UILog
}

// function newLayout() creates the initial layout of the user interface and
// populates it with the primary widgets.
func newLayout() *Layout {

	ui := tview.NewApplication()

	log := newUILog(ui)
	header := tview.NewBox()

	menu := tview.NewBox()
	main := tview.NewBox()
	sideBar := tview.NewBox()

	grid := tview.NewGrid().
		SetRows(3, 0, 8).
		SetColumns(32, 0, 32).
		SetBorders(true).
		AddItem(header, 0, 0, 1, 3, 0, 0, false).
		AddItem(log, 2, 0, 1, 3, 0, 0, false)

	// layout for screens narrower than 100 cells.
	grid.AddItem(menu, 0, 0, 0, 0, 0, 0, false).
		AddItem(main, 1, 0, 1, 3, 0, 0, false).
		AddItem(sideBar, 0, 0, 0, 0, 0, 0, false)

	// layout for screens wider than 100 cells.
	grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false).
		AddItem(main, 1, 1, 1, 1, 0, 100, false).
		AddItem(sideBar, 1, 2, 1, 1, 0, 100, false)

	ui.SetRoot(grid, true)
	ui.SetFocus(log)

	return &Layout{ui: ui, grid: grid, log: log}
}

// function show() starts drawing the user interface.
func (l *Layout) show() *ReturnCode {

	if err := l.ui.Run(); err != nil {
		return rcTUIError.specf("show(): ui.Run(): %s", err)
	}
	return nil
}

// type UIPrimitive defines the related fields describing any visual widget or
// component displayed in a Layout.
type UIPrimitive struct {
}

// function newUIPrimitive() allocates and initializes all default data for all
// widgets displayed to the user.
func newUIPrimitive() *UIPrimitive {
	return &UIPrimitive{}
}

// type UILog defines the tview.TextView widget where all runtime log data is
// navigated by and displayed to the user.00
type UILog struct {
	*UIPrimitive
	*tview.TextView
}

// function newUILog() allocates and initializes all default data for a runtime
// log view.
func newUILog(ui *tview.Application) *UILog {

	tvChanged := func() { ui.Draw() }
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
