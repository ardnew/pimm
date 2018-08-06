package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	MDName int = iota
	MDSize
	MDModTime
	MDLibrary
	MDSubtitles
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
	absolute   int
	isAbsolute bool
	isVisible  bool
	focusRune  rune
	media      *Media
}

func NewMediaDetailView(container *tview.Flex) *MediaDetailView {

	mediaDetailView := &MediaDetailView{
		Form:       tview.NewForm(),
		ui:         nil,
		obscura:    container,
		proportion: 0,
		absolute:   11,
		isAbsolute: true,
		isVisible:  true,
		focusRune:  MediaDetailFocusRune,
	}
	mediaDetailView.SetTitle(" Info (i) ")
	mediaDetailView.SetBorder(true)
	mediaDetailView.SetHorizontal(false)
	mediaDetailView.SetItemPadding(0)

	mediaDetailView.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	mediaDetailView.SetFieldTextColor(tcell.ColorWhite)
	mediaDetailView.SetLabelColor(tcell.ColorBlue)

	return mediaDetailView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *MediaDetailView) UI() *UI              { return view.ui.(*UI) }
func (view *MediaDetailView) FocusRune() rune      { return view.focusRune }
func (view *MediaDetailView) Obscura() *tview.Flex { return view.obscura }
func (view *MediaDetailView) Proportion() int      { return view.proportion }
func (view *MediaDetailView) Absolute() int        { return view.absolute }
func (view *MediaDetailView) IsAbsolute() bool     { return view.isAbsolute }
func (view *MediaDetailView) Visible() bool        { return view.isVisible }
func (view *MediaDetailView) Resizable() bool      { return true }

func (view *MediaDetailView) SetVisible(visible bool) {

	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, view.Absolute(), view.Proportion())
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
			infoLog.Log("+unhandled")
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

	view.Clear(true)

	view.Form.AddInputField("Name", media.Name(), 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	view.Form.AddInputField("Size", media.SizeStr(), 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	view.Form.AddInputField("ModTime", media.MTimeStr(), 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	view.Form.AddInputField("Library", media.library.Name(), 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	view.Form.AddInputField("Subtitles", "(sub)", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	view.Form.AddInputField("Type", "(ext)", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
	view.Form.AddInputField("Command", "(cmd)", 0,
		func(text string, last rune) bool { return false },
		func(text string) {})
}
