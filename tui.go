package main

import (
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type LibraryView struct {
	*tview.TreeView
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	size       int
}

type MediaView struct {
	*tview.Table
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	size       int
}

type MediaDetailView struct {
	*tview.Form
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	size       int
}

type PlaylistView struct {
	*tview.List
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	size       int
}

type LogView struct {
	*tview.TextView
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	size       int
}

type ModalView struct {
	*tview.Modal
}

type PageControl struct {
	*tview.Pages
	focusedView tview.Primitive
}

type MediaRef struct {
	library       *Library
	media         *Media
	libraryNode   *tview.TreeNode
	mediaCell     *tview.TableCell
	playlistIndex int
}

type UI struct {
	app *tview.Application

	sigDraw chan interface{}

	media map[*Library]*MediaRef

	focusLocked      bool
	focusLockedView  UIView
	focusTitleColor  map[bool]tcell.Color
	focusBorderColor map[bool]tcell.Color

	libraryView     *LibraryView
	mediaView       *MediaView
	mediaDetailView *MediaDetailView
	playlistView    *PlaylistView
	logView         *LogView
	pageControl     *PageControl
}

type UIView interface {
	RootUI() *UI
	FocusRune() rune
	LockFocus(bool)
	Visible() bool
	SetVisible(bool)
	Proportion() int
	Obscura() *tview.Flex
}

const (
	PageIDBrowser   = "browse"
	PageIDQuitModal = "quit"

	LibraryFocusRune     = 'l'
	MediaFocusRune       = 'm'
	MediaDetailFocusRune = 'i'
	PlaylistFocusRune    = 'p'
	LogFocusRune         = 'v'
	QuitRune             = 'q'
)

type ViewPrimitive struct {
	view UIView
	prim tview.Primitive
}

var (
	self         *tview.Application
	focusKeyView = map[rune]ViewPrimitive{}
)

func NewUI(opt *Options) *UI {

	self = tview.NewApplication()
	sigDraw := make(chan interface{})

	//
	// containers for the primitive views
	//
	mediaRowsLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mediaRowsLayout.SetBorder(true)
	mediaRowsLayout.SetTitle(" Media (m) ")

	browseColsLayout := tview.NewFlex().SetDirection(tview.FlexColumn)

	browseMainLayout := tview.NewFlex().SetDirection(tview.FlexRow)

	//
	// library tree view
	//
	libraryView := &LibraryView{
		TreeView:   tview.NewTreeView(),
		rootUI:     nil,
		obscura:    browseColsLayout,
		proportion: 1,
		isVisible:  true,
		focusRune:  LibraryFocusRune,
	}
	libraryView.SetTitle(" Library (l) ")
	libraryView.SetBorder(true)
	libraryView.SetGraphics(true)
	libraryView.SetTopLevel(1)

	//
	// media table view
	//
	mediaView := &MediaView{
		Table:      tview.NewTable(),
		rootUI:     nil,
		obscura:    mediaRowsLayout,
		proportion: 3,
		isVisible:  true,
		focusRune:  MediaFocusRune,
	}
	mediaView.SetTitle("")
	mediaView.SetBorder(false)
	mediaView.SetBorders(false)
	mediaView.SetSelectable(true /*rows*/, false /*cols*/)

	//
	// embedded selected-media-info form view
	//
	mediaDetailView := &MediaDetailView{
		Form:       tview.NewForm(),
		rootUI:     nil,
		obscura:    mediaRowsLayout,
		proportion: 1,
		isVisible:  true,
		focusRune:  MediaDetailFocusRune,
	}
	mediaDetailView.SetTitle(" Info (i) ")
	mediaDetailView.SetBorder(true)
	mediaDetailView.SetHorizontal(false)

	//
	// playlist list view
	//
	playlistView := &PlaylistView{
		List:       tview.NewList(),
		rootUI:     nil,
		obscura:    browseColsLayout,
		proportion: 1,
		isVisible:  true,
		focusRune:  PlaylistFocusRune,
	}
	playlistView.SetTitle(" Playlist (p) ")
	playlistView.SetBorder(true)
	playlistView.ShowSecondaryText(false)

	//
	// active log text view
	//
	logView := &LogView{
		TextView:   tview.NewTextView(),
		rootUI:     nil,
		obscura:    browseMainLayout,
		proportion: 1,
		isVisible:  true,
		focusRune:  LogFocusRune,
	}
	logView.SetTitle(" Log (v) ")
	logView.SetBorder(true)
	logView.SetDynamicColors(true)
	logView.SetRegions(true)
	logView.SetScrollable(true)
	logView.SetWrap(false)
	setLogWriter(logView)

	//
	// composition of container views
	//
	mediaRowsLayout.AddItem(mediaView, 0, mediaView.Proportion(), false)
	mediaRowsLayout.AddItem(mediaDetailView, 0, mediaDetailView.Proportion(), false)

	browseColsLayout.AddItem(libraryView, 0, libraryView.Proportion(), false)
	browseColsLayout.AddItem(mediaRowsLayout, 0, 2, false)
	browseColsLayout.AddItem(playlistView, 0, playlistView.Proportion(), false)

	browseMainLayout.AddItem(browseColsLayout, 0, 3, false)
	browseMainLayout.AddItem(logView, 0, logView.Proportion(), false)

	//
	// root-level container selects which app page to display
	//
	pageControl := &PageControl{
		Pages:       tview.NewPages(),
		focusedView: mediaView,
	}
	pageControl.AddPage(PageIDBrowser, browseMainLayout, true, true)

	//
	// construct reference table for focus key handlers
	//
	focusKeyView[libraryView.FocusRune()] =
		ViewPrimitive{libraryView, libraryView}
	focusKeyView[mediaView.FocusRune()] =
		ViewPrimitive{mediaView, mediaView}
	focusKeyView[mediaDetailView.FocusRune()] =
		ViewPrimitive{mediaDetailView, mediaDetailView}
	focusKeyView[playlistView.FocusRune()] =
		ViewPrimitive{playlistView, playlistView}
	focusKeyView[logView.FocusRune()] =
		ViewPrimitive{logView, logView}

	//
	// layout definition of entire UI
	//
	rootUI := &UI{
		app: self,

		sigDraw: sigDraw,

		media: make(map[*Library]*MediaRef),

		focusLocked:     false,
		focusLockedView: nil,
		focusTitleColor: map[bool]tcell.Color{
			false: tcell.ColorWhite,
			true:  tcell.ColorDeepSkyBlue,
		},
		focusBorderColor: map[bool]tcell.Color{
			false: tcell.ColorWhite,
			true:  tcell.ColorSteelBlue,
		},

		libraryView:     libraryView,
		mediaView:       mediaView,
		mediaDetailView: mediaDetailView,
		playlistView:    playlistView,
		logView:         logView,
		pageControl:     pageControl,
	}

	//
	// initially focused view
	//
	self.SetRoot(pageControl, true)
	self.SetFocus(mediaView)
	mediaView.obscura.SetTitleColor(rootUI.focusTitleColor[true])
	mediaView.obscura.SetBorderColor(rootUI.focusBorderColor[true])

	//
	// backreference so each view has easy access to every other
	//
	rootUI.libraryView.rootUI = rootUI
	rootUI.mediaView.rootUI = rootUI
	rootUI.mediaDetailView.rootUI = rootUI
	rootUI.playlistView.rootUI = rootUI
	rootUI.logView.rootUI = rootUI

	return rootUI
}

func isQuitEventKeyRune(keyRune rune) bool { return QuitRune == keyRune }

func (ui *UI) ConfirmQuitPrompt() {

	modalView := &ModalView{Modal: tview.NewModal()}
	modalView.SetText("Calling it quits?")
	modalView.AddButtons([]string{"Quit", "Cancel"})
	modalView.SetDoneFunc(
		func(buttonIndex int, buttonLabel string) {
			switch {
			case 0 == buttonIndex: // "Quit"
				ui.app.Stop()
			case 1 == buttonIndex || buttonIndex < 0: // "Cancel"/ESC pressed
				ui.pageControl.RemovePage(PageIDQuitModal)
				ui.app.SetFocus(ui.pageControl.focusedView)
			}
		})

	ui.pageControl.AddPage(PageIDQuitModal, modalView, true, true)
}

func (ui *UI) GlobalInputHandled(
	view UIView, event *tcell.EventKey, setFocus func(p tview.Primitive)) bool {

	inKey := event.Key()
	inMods := event.Modifiers()
	//inName := event.Name()
	inRune := event.Rune()
	//inWhen := event.When()

	switch inMods {

	case tcell.ModShift:
	case tcell.ModCtrl:

		switch inKey {

		case tcell.KeyCtrlC:
			ui.ConfirmQuitPrompt()
			return true
		}

	case tcell.ModAlt:

		switch target, ok := focusKeyView[inRune]; {

		case ok:
			target.view.SetVisible(!target.view.Visible())

		}

	case tcell.ModMeta:
	case tcell.ModNone:

		switch inKey {

		case tcell.KeyEscape:
			if ui.pageControl.focusedView.(UIView).Visible() {
				ui.pageControl.focusedView.(UIView).LockFocus(!ui.focusLocked)
			}

		case tcell.KeyRune:

			switch {
			case isQuitEventKeyRune(inRune):
				ui.ConfirmQuitPrompt()
				return true

			default:
				if target, ok := focusKeyView[inRune]; ok {
					if !ui.focusLocked {
						infoLog.Logf("focused: %T", target.prim)
						ui.pageControl.focusedView = target.prim
						setFocus(target.prim)
						target.view.SetVisible(true)
						return true
					}
				}
			}

		default:
			// nothing
		}

	}

	return false
}

func (view *LibraryView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.TreeView.InputHandler()(event, setFocus)
		})
}

func (view *MediaView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Table.InputHandler()(event, setFocus)
		})
}

func (view *MediaDetailView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Form.InputHandler()(event, setFocus)
		})
}

func (view *PlaylistView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.List.InputHandler()(event, setFocus)
		})
}

func (view *LogView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			handled := false
			switch event.Key() {
			case tcell.KeyRune:
				if ' ' == event.Rune() {
					for n, c := range tcell.ColorNames {
						v := tcell.ColorValues[c]
						s := fmt.Sprintf("[#%06x] %24s: %10d 0x%08x [-:-:-]", v, n, v, v)
						infoLog.Log(s)
					}
					handled = true
				}
			}
			if !handled {
				view.TextView.InputHandler()(event, setFocus)
			}
		})
}

func (view *LibraryView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[true])
		view.SetBorderColor(view.RootUI().focusBorderColor[true])
	}
	view.TreeView.Focus(delegate)
}
func (view *LibraryView) Blur() {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[false])
		view.SetBorderColor(view.RootUI().focusBorderColor[false])
	}
	view.TreeView.Blur()
}

func (view *MediaView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.rootUI {
		view.Obscura().SetTitleColor(view.RootUI().focusTitleColor[true])
		view.Obscura().SetBorderColor(view.RootUI().focusBorderColor[true])
	}
	view.Table.Focus(delegate)
}
func (view *MediaView) Blur() {
	if nil != view.rootUI {
		view.Obscura().SetTitleColor(view.RootUI().focusTitleColor[false])
		view.Obscura().SetBorderColor(view.RootUI().focusBorderColor[false])
	}
	view.Table.Blur()
}

func (view *MediaDetailView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[true])
		view.SetBorderColor(view.RootUI().focusBorderColor[true])
	}
	view.Form.Focus(delegate)
}
func (view *MediaDetailView) Blur() {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[false])
		view.SetBorderColor(view.RootUI().focusBorderColor[false])
	}
	view.Form.Blur()
}

func (view *PlaylistView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[true])
		view.SetBorderColor(view.RootUI().focusBorderColor[true])
	}
	view.List.Focus(delegate)
}
func (view *PlaylistView) Blur() {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[false])
		view.SetBorderColor(view.RootUI().focusBorderColor[false])
	}
	view.List.Blur()
}

func (view *LogView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[true])
		view.SetBorderColor(view.RootUI().focusBorderColor[true])
	}
	view.TextView.Focus(delegate)
}
func (view *LogView) Blur() {
	if nil != view.rootUI {
		view.SetTitleColor(view.RootUI().focusTitleColor[false])
		view.SetBorderColor(view.RootUI().focusBorderColor[false])
	}
	view.TextView.Blur()
}

// converts the rootUI interface to a concrete *UI type
func (view *LibraryView) RootUI() *UI     { return view.rootUI.(*UI) }
func (view *MediaView) RootUI() *UI       { return view.rootUI.(*UI) }
func (view *MediaDetailView) RootUI() *UI { return view.rootUI.(*UI) }
func (view *PlaylistView) RootUI() *UI    { return view.rootUI.(*UI) }
func (view *LogView) RootUI() *UI         { return view.rootUI.(*UI) }

func (view *LibraryView) FocusRune() rune     { return LibraryFocusRune }
func (view *MediaView) FocusRune() rune       { return MediaFocusRune }
func (view *MediaDetailView) FocusRune() rune { return MediaDetailFocusRune }
func (view *PlaylistView) FocusRune() rune    { return PlaylistFocusRune }
func (view *LogView) FocusRune() rune         { return LogFocusRune }

func (view *LibraryView) LockFocus(lock bool) {
	view.RootUI().focusLocked = lock
	view.RootUI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.RootUI().focusBorderColor[view.RootUI().pageControl.focusedView == view])
	}
}
func (view *MediaView) LockFocus(lock bool) {
	view.RootUI().focusLocked = lock
	view.RootUI().focusLockedView = view
	if lock {
		view.Obscura().SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.Obscura().SetBorderColor(view.RootUI().focusBorderColor[view.RootUI().pageControl.focusedView == view])
	}
}
func (view *MediaDetailView) LockFocus(lock bool) {
	view.RootUI().focusLocked = lock
	view.RootUI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.RootUI().focusBorderColor[view.RootUI().pageControl.focusedView == view])
	}
}
func (view *PlaylistView) LockFocus(lock bool) {
	view.RootUI().focusLocked = lock
	view.RootUI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.RootUI().focusBorderColor[view.RootUI().pageControl.focusedView == view])
	}
}
func (view *LogView) LockFocus(lock bool) {
	view.RootUI().focusLocked = lock
	view.RootUI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.RootUI().focusBorderColor[view.RootUI().pageControl.focusedView == view])
	}
}

func (view *LibraryView) Visible() bool     { return view.isVisible }
func (view *MediaView) Visible() bool       { return view.isVisible }
func (view *MediaDetailView) Visible() bool { return view.isVisible }
func (view *PlaylistView) Visible() bool    { return view.isVisible }
func (view *LogView) Visible() bool         { return view.isVisible }

func (view *LibraryView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if visible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.RootUI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

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
			view.RootUI().mediaDetailView.SetVisible(true)
			if view.RootUI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *MediaDetailView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.RootUI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *PlaylistView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.RootUI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *LogView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.RootUI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *LibraryView) Proportion() int     { return view.proportion }
func (view *MediaView) Proportion() int       { return view.proportion }
func (view *MediaDetailView) Proportion() int { return view.proportion }
func (view *PlaylistView) Proportion() int    { return view.proportion }
func (view *LogView) Proportion() int         { return view.proportion }

func (view *LibraryView) Obscura() *tview.Flex     { return view.obscura }
func (view *MediaView) Obscura() *tview.Flex       { return view.obscura }
func (view *MediaDetailView) Obscura() *tview.Flex { return view.obscura }
func (view *PlaylistView) Obscura() *tview.Flex    { return view.obscura }
func (view *LogView) Obscura() *tview.Flex         { return view.obscura }
