package main

import (
	"sync"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	PopulateCycleDuration = 10 * time.Millisecond
)

type MediaView struct {
	*tview.Table
	sync.Mutex
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
}

func NewMediaView(container *tview.Flex) *MediaView {

	mediaView := &MediaView{
		Table:      tview.NewTable(),
		Mutex:      sync.Mutex{},
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
	mediaView.SetBorderColor(tcell.ColorDarkGray)
	mediaView.SetBorders(false)
	mediaView.SetSelectable(true /*rows*/, false /*cols*/)

	return mediaView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *MediaView) UI() *UI              { return view.ui.(*UI) }
func (view *MediaView) FocusRune() rune      { return view.focusRune }
func (view *MediaView) Obscura() *tview.Flex { return view.obscura }
func (view *MediaView) Proportion() int      { return view.proportion }
func (view *MediaView) Visible() bool        { return view.isVisible }
func (view *MediaView) Resizable() bool      { return false }

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

func (view *MediaView) LockFocus(lock bool) {

	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.Obscura().SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.Obscura().SetBorderColor(
			view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

// -----------------------------------------------------------------------------
//  (tview) embedded Primitive.(Table)
// -----------------------------------------------------------------------------

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

func (view *MediaView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Table.InputHandler()(event, setFocus)
		})
}

// -----------------------------------------------------------------------------
//  (pimm) MediaView
// -----------------------------------------------------------------------------

func (view *MediaView) appendMedia(media *Media) {

	nameCell := tview.NewTableCell(media.Name())
	libCell := tview.NewTableCell(media.library.Name())
	dateCell := tview.NewTableCell(media.MTimeStr())

	view.Lock()
	row := view.GetRowCount()
	view.SetCell(row, 0, nameCell)
	view.SetCell(row, 1, libCell)
	view.SetCell(row, 2, dateCell)
	view.Unlock()
	//view.UI().app.Draw()
}

func (view *MediaView) AddMedia(media *Media) {

	//view.SetCell(view.numRows, 0, nameCell)
	//view.SetCell(view.numRows, 1, libCell)
	//view.SetCell(view.numRows, 2, dateCell)
	//view.cellChan <- &CellPosition{view.numRows, 0, nameCell}
	//view.cellChan <- &CellPosition{view.numRows, 1, dateCell}

	view.appendMedia(media)
}
