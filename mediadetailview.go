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
	MDCOUNT
)

type DetailFormItem struct {
	*tview.InputField
	item   int
	parent *MediaDetailView
}

func NewDetailFormItem(parent *MediaDetailView, item int, name string) *DetailFormItem {

	field := tview.NewInputField()
	field.SetLabel(name)

	field.SetAcceptanceFunc(func(text string, lastChar rune) bool { return false })
	field.SetChangedFunc(nil)
	field.SetDoneFunc(nil)
	field.SetFinishedFunc(nil)

	detail := &DetailFormItem{
		InputField: field,
		item:       item,
		parent:     parent,
	}

	return detail
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
	detail     [MDCOUNT]*DetailFormItem
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
		detail:     [MDCOUNT]*DetailFormItem{},
	}
	mediaDetailView.SetTitle(" Info (i) ")
	mediaDetailView.SetBorder(true)
	mediaDetailView.SetHorizontal(false)
	mediaDetailView.SetItemPadding(0)

	mediaDetailView.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	mediaDetailView.SetFieldTextColor(tcell.ColorWhite)
	mediaDetailView.SetLabelColor(tcell.ColorBlue)

	mediaDetailView.detail[MDName] = NewDetailFormItem(mediaDetailView, MDName, "Name")
	mediaDetailView.detail[MDSize] = NewDetailFormItem(mediaDetailView, MDSize, "Size")
	mediaDetailView.detail[MDModTime] = NewDetailFormItem(mediaDetailView, MDModTime, "ModTime")
	mediaDetailView.detail[MDLibrary] = NewDetailFormItem(mediaDetailView, MDLibrary, "Library")
	mediaDetailView.detail[MDSubtitles] = NewDetailFormItem(mediaDetailView, MDSubtitles, "Subtitles")
	mediaDetailView.detail[MDType] = NewDetailFormItem(mediaDetailView, MDType, "Type")
	mediaDetailView.detail[MDCommand] = NewDetailFormItem(mediaDetailView, MDCommand, "Command")

	for _, d := range mediaDetailView.detail {
		mediaDetailView.AddFormItem(d)
	}

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
		view.SetBorderColor(view.UI().focusLockedColor)
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
//  (tview) embedded Primitive.(InputField)
// -----------------------------------------------------------------------------

func (view *DetailFormItem) Focus(delegate func(p tview.Primitive)) {

	locked := view.parent.UI().focusLockedView

	if nil != view.parent && locked != view.parent {
		view.parent.SetTitleColor(view.parent.UI().focusTitleColor[true])
		view.parent.SetBorderColor(view.parent.UI().focusBorderColor[true])
	}
	view.InputField.Focus(delegate)
}

func (view *DetailFormItem) Blur() {

	//locked := view.parent.UI().focusLockedView

	if nil != view.parent {
		view.parent.SetTitleColor(view.parent.UI().focusTitleColor[false])
		view.parent.SetBorderColor(view.parent.UI().focusBorderColor[false])
	}
	view.InputField.Blur()
}

func (view *DetailFormItem) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

	isTableNavKey := func(event *tcell.EventKey) bool {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
			return true
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h', 'l', 'j', 'k', 'g', 'G':
				return true
			}
		}
		return false
	}

	return view.parent.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.parent.UI().GlobalInputHandled(view.parent, event, setFocus) {
				return
			}
			if isTableNavKey(event) && !view.parent.UI().focusLocked {
				// never forward the keystrokes anywhere if we have a focus lock
				view.parent.UI().mediaView.InputHandler()(event, setFocus)
				return
			}
			switch event.Key() {
			case tcell.KeyEnter, tcell.KeyEscape:
				return
			default:
				view.InputField.InputHandler()(event, setFocus)
			}
		})
}

// -----------------------------------------------------------------------------
//  (pimm) DetailFormItem
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
//  (pimm) MediaDetailView
// -----------------------------------------------------------------------------

func (view *MediaDetailView) SetMedia(media *Media) {

	view.media = media

	view.detail[MDName].SetText(media.Name())
	view.detail[MDSize].SetText(media.SizeStr())
	view.detail[MDModTime].SetText(media.MTimeStr())
	view.detail[MDLibrary].SetText(media.library.Name())
	view.detail[MDSubtitles].SetText("(sub)")
	view.detail[MDType].SetText("(ext)")
	view.detail[MDCommand].SetText("(cmd)")
}
