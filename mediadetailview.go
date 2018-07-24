package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	MDLibrary int = iota
	MDName
	MDSize
	MDModTime
	MDType
	MDCommand
)

type DetailFormItem struct {
	tview.InputField
}

type MediaDetailView struct {
	*tview.Form
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	media      *Media
}

func NewMediaDetailView(container *tview.Flex) *MediaDetailView {

	mediaDetailView := &MediaDetailView{
		Form:       tview.NewForm(),
		ui:         nil,
		obscura:    container,
		proportion: 1,
		isVisible:  true,
		focusRune:  MediaDetailFocusRune,
	}
	mediaDetailView.SetTitle(" Info (i) ")
	mediaDetailView.SetBorder(true)
	mediaDetailView.SetHorizontal(false)

	mediaDetailView.Form.AddInputField("Library", "", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	mediaDetailView.Form.AddInputField("Name", "", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	mediaDetailView.Form.AddInputField("Size", "", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	mediaDetailView.Form.AddInputField("ModTime", "", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	mediaDetailView.Form.AddInputField("Type", "", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	mediaDetailView.Form.AddInputField("Command", "", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})

	return mediaDetailView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *MediaDetailView) UI() *UI              { return view.ui.(*UI) }
func (view *MediaDetailView) FocusRune() rune      { return view.focusRune }
func (view *MediaDetailView) Obscura() *tview.Flex { return view.obscura }
func (view *MediaDetailView) Proportion() int      { return view.proportion }
func (view *MediaDetailView) Visible() bool        { return view.isVisible }
func (view *MediaDetailView) Resizable() bool      { return true }

func (view *MediaDetailView) SetVisible(visible bool) {

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

func (view *MediaDetailView) LockFocus(lock bool) {

	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

// -----------------------------------------------------------------------------
//  (tview) embedded Primitive.(Form)
// -----------------------------------------------------------------------------

func (view *MediaDetailView) Focus(delegate func(p tview.Primitive)) {

	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[true])
		view.SetBorderColor(view.UI().focusBorderColor[true])
	}
	view.Form.Focus(delegate)
}

func (view *MediaDetailView) Blur() {

	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[false])
		view.SetBorderColor(view.UI().focusBorderColor[false])
	}
	view.Form.Blur()
}

func (view *MediaDetailView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Form.InputHandler()(event, setFocus)
		})
}

// -----------------------------------------------------------------------------
//  (pimm) MediaDetailView
// -----------------------------------------------------------------------------

func (view *MediaDetailView) SetMedia(media *Media) {

	view.media = media

	view.Form.GetFormItem(MDLibrary)
	view.Form.GetFormItem(MDName)
	view.Form.GetFormItem(MDSize)
	view.Form.GetFormItem(MDModTime)
	view.Form.GetFormItem(MDType)
	view.Form.GetFormItem(MDCommand)
}
