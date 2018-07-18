package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type PlaylistView struct {
	*tview.List
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
}

func NewPlaylistView(container *tview.Flex) *PlaylistView {

	playlistView := &PlaylistView{
		List:       tview.NewList(),
		ui:         nil,
		obscura:    container,
		proportion: 1,
		isVisible:  true,
		focusRune:  PlaylistFocusRune,
	}
	playlistView.SetTitle(" Playlist (p) ")
	playlistView.SetBorder(true)
	playlistView.ShowSecondaryText(false)

	return playlistView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *PlaylistView) UI() *UI              { return view.ui.(*UI) }
func (view *PlaylistView) FocusRune() rune      { return view.focusRune }
func (view *PlaylistView) Obscura() *tview.Flex { return view.obscura }
func (view *PlaylistView) Proportion() int      { return view.proportion }
func (view *PlaylistView) Visible() bool        { return view.isVisible }

func (view *PlaylistView) SetVisible(visible bool) {

	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.UI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *PlaylistView) LockFocus(lock bool) {

	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

// -----------------------------------------------------------------------------
//  (tview) embedded Primitive.(List)
// -----------------------------------------------------------------------------

func (view *PlaylistView) Focus(delegate func(p tview.Primitive)) {

	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[true])
		view.SetBorderColor(view.UI().focusBorderColor[true])
	}
	view.List.Focus(delegate)
}

func (view *PlaylistView) Blur() {

	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[false])
		view.SetBorderColor(view.UI().focusBorderColor[false])
	}
	view.List.Blur()
}

func (view *PlaylistView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.List.InputHandler()(event, setFocus)
		})
}

// -----------------------------------------------------------------------------
//  (pimm) PlaylistView
// -----------------------------------------------------------------------------
