package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type MediaView struct {
	*tview.Table
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
}

func NewMediaView(container *tview.Flex) *MediaView {

	mediaView := &MediaView{
		Table:      tview.NewTable(),
		ui:         nil,
		obscura:    container,
		proportion: 3,
		isVisible:  true,
		focusRune:  MediaFocusRune,
	}
	container.SetTitle(" Media (m) ")
	container.SetBorder(true)
	mediaView.SetTitle("")
	mediaView.SetBorder(false)
	mediaView.SetBorders(false)
	mediaView.SetSelectable(true /*rows*/, false /*cols*/)

	return mediaView
}

func (view *MediaView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Table.InputHandler()(event, setFocus)
		})
}

func (view *MediaView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.ui {
		view.Obscura().SetTitleColor(view.UI().focusTitleColor[true])
		view.Obscura().SetBorderColor(view.UI().focusBorderColor[true])
	}
	view.Table.Focus(delegate)
}

func (view *MediaView) Blur() {
	if nil != view.ui {
		view.Obscura().SetTitleColor(view.UI().focusTitleColor[false])
		view.Obscura().SetBorderColor(view.UI().focusBorderColor[false])
	}
	view.Table.Blur()
}

func (view *MediaView) UI() *UI              { return view.ui.(*UI) }
func (view *MediaView) FocusRune() rune      { return view.focusRune }
func (view *MediaView) Obscura() *tview.Flex { return view.obscura }
func (view *MediaView) Proportion() int      { return view.proportion }

func (view *MediaView) LockFocus(lock bool) {
	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.Obscura().SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.Obscura().SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

func (view *MediaView) Visible() bool { return view.isVisible }
func (view *MediaView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			// if the media table is collapsing, fill remaining space with
			// the media info form
			view.UI().mediaDetailView.SetVisible(true)
			if view.UI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}
