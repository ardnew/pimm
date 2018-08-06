package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.Grid
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	absolute   int
	isAbsolute bool
	isVisible  bool
	focusRune  rune
}

func NewHelpView(container *tview.Flex) *HelpView {

	helpView := &HelpView{
		Grid:       tview.NewGrid(),
		ui:         nil,
		obscura:    container,
		proportion: 0,
		absolute:   10,
		isAbsolute: true,
		isVisible:  true,
		focusRune:  HelpRune,
	}
	helpView.SetTitle(" Help (h) ")
	helpView.SetTitleAlign(tview.AlignLeft)
	helpView.SetBorder(true)

	return helpView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *HelpView) UI() *UI              { return view.ui.(*UI) }
func (view *HelpView) FocusRune() rune      { return view.focusRune }
func (view *HelpView) Obscura() *tview.Flex { return view.obscura }
func (view *HelpView) Proportion() int      { return view.proportion }
func (view *HelpView) Absolute() int        { return view.absolute }
func (view *HelpView) IsAbsolute() bool     { return view.isAbsolute }
func (view *HelpView) Visible() bool        { return view.isVisible }
func (view *HelpView) Resizable() bool      { return true }

func (view *HelpView) SetVisible(visible bool) {

	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			view.SetBorderColor(tcell.ColorWheat)
			view.SetTitleColor(tcell.ColorOrange)
			obs.ResizeItem(view, view.Absolute(), view.Proportion())
		} else {
			color, _ := ColorTransparent()
			view.SetBorderColor(color)
			view.SetTitleColor(tcell.ColorOrange)
			obs.ResizeItem(view, 2, 0)
			if view.UI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *HelpView) LockFocus(lock bool) {

	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.SetBorderColor(view.UI().focusLockedColor)
	} else {
		view.SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

// -----------------------------------------------------------------------------
//  (tview) embedded Primitive.(Grid)
// -----------------------------------------------------------------------------

func (view *HelpView) Focus(delegate func(p tview.Primitive)) {

	view.Grid.Focus(delegate)
}

func (view *HelpView) Blur() {

	view.Grid.Blur()
}

func (view *HelpView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Grid.InputHandler()(event, setFocus)
		})
}

// -----------------------------------------------------------------------------
//  (pimm) HelpView
// -----------------------------------------------------------------------------
