package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type LibraryView struct {
	*tview.TreeView
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
}

type MediaView struct {
	*tview.Table
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
}

type MediaDetailView struct {
	*tview.Form
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
}

type PlaylistView struct {
	*tview.List
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
}

type LogView struct {
	*tview.TextView
	rootUI     interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
}

type UIInputHandler func(event *tcell.EventKey, setFocus func(p tview.Primitive))

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
	focusBorderColor map[bool]tcell.Color

	libraryView     *LibraryView
	mediaView       *MediaView
	mediaDetailView *MediaDetailView
	playlistView    *PlaylistView
	logView         *LogView
}

type UIView interface {
	RootUI() *UI
	FocusKeyRune() rune
	LockFocus(lock bool)
	Visible() bool
	SetVisible(bool)
	Proportion() int
	Obscura() *tview.Flex
}

var (
	self            *tview.Application
	focusKeyHandler = map[rune]tview.Primitive{}
)

func NewUI(opt *Options) *UI {

	self = tview.NewApplication()
	sigDraw := make(chan interface{})

	libraryRowsLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mediaRowsLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	playlistRowsLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	browseColsLayout := tview.NewFlex().SetDirection(tview.FlexColumn)
	browseMainLayout := tview.NewFlex().SetDirection(tview.FlexRow)

	libraryView := &LibraryView{
		TreeView:   tview.NewTreeView(),
		rootUI:     nil,
		obscura:    browseColsLayout,
		proportion: 1,
		isVisible:  true,
	}
	libraryView.SetTitle(" Library ")
	libraryView.SetBorder(true)
	libraryView.SetGraphics(true)
	libraryView.SetTopLevel(1)

	mediaView := &MediaView{
		Table:      tview.NewTable(),
		rootUI:     nil,
		obscura:    mediaRowsLayout,
		proportion: 3,
		isVisible:  true,
	}
	mediaView.SetTitle("")
	mediaView.SetBorder(false)
	mediaView.SetBorders(false)
	mediaView.SetSelectable(true /*rows*/, false /*cols*/)

	mediaDetailView := &MediaDetailView{
		Form:       tview.NewForm(),
		rootUI:     nil,
		obscura:    mediaRowsLayout,
		proportion: 1,
		isVisible:  true,
	}
	mediaDetailView.SetTitle(" Info ")
	mediaDetailView.SetBorder(true)
	mediaDetailView.SetHorizontal(false)

	mediaRowsLayout.SetBorder(true)
	mediaRowsLayout.SetTitle(" Media ")

	playlistView := &PlaylistView{
		List:       tview.NewList(),
		rootUI:     nil,
		obscura:    browseColsLayout,
		proportion: 1,
		isVisible:  true,
	}
	playlistView.SetTitle(" Playlist ")
	playlistView.SetBorder(true)
	playlistView.ShowSecondaryText(false)

	logView := &LogView{
		TextView:   tview.NewTextView(),
		rootUI:     nil,
		obscura:    mediaRowsLayout,
		proportion: 1,
		isVisible:  true,
	}
	logView.SetTitle(" Log ")
	logView.SetBorder(true)
	logView.SetDynamicColors(true)
	logView.SetRegions(true)
	logView.SetScrollable(true)
	logView.SetWrap(false)
	setLogWriter(logView)

	libraryRowsLayout.AddItem(libraryView, 0, libraryView.Proportion(), false)

	mediaRowsLayout.AddItem(mediaView, 0, mediaView.Proportion(), false)
	mediaRowsLayout.AddItem(mediaDetailView, 0, mediaDetailView.Proportion(), false)

	playlistRowsLayout.AddItem(playlistView, 0, playlistView.Proportion(), false)

	browseColsLayout.AddItem(libraryRowsLayout, 0, 1, false)
	browseColsLayout.AddItem(mediaRowsLayout, 0, 2, false)
	browseColsLayout.AddItem(playlistRowsLayout, 0, 1, false)

	browseMainLayout.AddItem(browseColsLayout, 0, 3, false)
	browseMainLayout.AddItem(logView, 0, logView.Proportion(), false)

	pageLayout := tview.NewPages()
	pageLayout.AddPage("browser", browseMainLayout, true, true)

	self.SetRoot(pageLayout, true)
	self.SetFocus(mediaView)

	focusKeyHandler[libraryView.FocusKeyRune()] = libraryView
	focusKeyHandler[mediaView.FocusKeyRune()] = mediaView
	focusKeyHandler[mediaDetailView.FocusKeyRune()] = mediaDetailView
	focusKeyHandler[playlistView.FocusKeyRune()] = playlistView
	focusKeyHandler[logView.FocusKeyRune()] = logView

	rootUI := &UI{
		app: self,

		sigDraw: sigDraw,

		media: make(map[*Library]*MediaRef),

		focusLocked: false,
		focusBorderColor: map[bool]tcell.Color{
			false: tcell.ColorWhite,
			true:  tcell.ColorGreen,
		},

		libraryView:     libraryView,
		mediaView:       mediaView,
		mediaDetailView: mediaDetailView,
		playlistView:    playlistView,
		logView:         logView,
	}

	// backreferences so each view has easy access to every other
	rootUI.libraryView.rootUI = rootUI
	rootUI.mediaView.rootUI = rootUI
	rootUI.mediaDetailView.rootUI = rootUI
	rootUI.playlistView.rootUI = rootUI
	rootUI.logView.rootUI = rootUI

	return rootUI
}

func (rootUI *UI) FocusInputHandler(view UIView, event *tcell.EventKey, setFocus func(p tview.Primitive)) bool {
	switch event.Key() {
	case tcell.KeyESC:
		view.LockFocus(!rootUI.focusLocked)
		return true

	case tcell.KeyCtrlP:
		rootUI.playlistView.SetVisible(!rootUI.playlistView.Visible())
		return true

	case tcell.KeyRune:
		if !rootUI.focusLocked {
			if focusView, handled := focusKeyHandler[event.Rune()]; handled {
				setFocus(focusView)
				return true
			}
		}
	}
	return false
}

func (view *LibraryView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().FocusInputHandler(view, event, setFocus) {
				return
			}
			view.TreeView.InputHandler()(event, setFocus)
		})
}

func (view *MediaView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().FocusInputHandler(view, event, setFocus) {
				return
			}
			view.Table.InputHandler()(event, setFocus)
		})
}

func (view *MediaDetailView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().FocusInputHandler(view, event, setFocus) {
				return
			}
			view.Form.InputHandler()(event, setFocus)
		})
}

func (view *PlaylistView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().FocusInputHandler(view, event, setFocus) {
				return
			}
			view.List.InputHandler()(event, setFocus)
		})
}

func (view *LogView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.RootUI().FocusInputHandler(view, event, setFocus) {
				return
			}
			view.TextView.InputHandler()(event, setFocus)
		})
}

// converts the rootUI interface to a concrete *UI type
func (view *LibraryView) RootUI() *UI     { return view.rootUI.(*UI) }
func (view *MediaView) RootUI() *UI       { return view.rootUI.(*UI) }
func (view *MediaDetailView) RootUI() *UI { return view.rootUI.(*UI) }
func (view *PlaylistView) RootUI() *UI    { return view.rootUI.(*UI) }
func (view *LogView) RootUI() *UI         { return view.rootUI.(*UI) }

func (view *LibraryView) FocusKeyRune() rune     { return 'l' }
func (view *MediaView) FocusKeyRune() rune       { return 'm' }
func (view *MediaDetailView) FocusKeyRune() rune { return 'i' }
func (view *PlaylistView) FocusKeyRune() rune    { return 'p' }
func (view *LogView) FocusKeyRune() rune         { return 'v' }

func (view *LibraryView) LockFocus(lock bool) {
	rootUI := view.RootUI()
	rootUI.focusLocked = lock
	view.SetBorderColor(rootUI.focusBorderColor[lock])
	infoLog.Logf("focus locked: %s", map[bool]string{false: "none", true: "LibraryView"}[rootUI.focusLocked])
}

func (view *MediaView) LockFocus(lock bool) {
	rootUI := view.RootUI()
	rootUI.focusLocked = lock
	view.Obscura().SetBorderColor(rootUI.focusBorderColor[lock])
	infoLog.Logf("focus locked: %s", map[bool]string{false: "none", true: "MediaView"}[rootUI.focusLocked])
}

func (view *MediaDetailView) LockFocus(lock bool) {
	rootUI := view.RootUI()
	rootUI.focusLocked = lock
	view.SetBorderColor(rootUI.focusBorderColor[lock])
	infoLog.Logf("focus locked: %s", map[bool]string{false: "none", true: "MediaDetailView"}[rootUI.focusLocked])
}

func (view *PlaylistView) LockFocus(lock bool) {
	rootUI := view.RootUI()
	rootUI.focusLocked = lock
	view.SetBorderColor(rootUI.focusBorderColor[lock])
	infoLog.Logf("focus locked: %s", map[bool]string{false: "none", true: "PlaylistView"}[rootUI.focusLocked])
}

func (view *LogView) LockFocus(lock bool) {
	rootUI := view.RootUI()
	rootUI.focusLocked = lock
	view.SetBorderColor(rootUI.focusBorderColor[lock])
	infoLog.Logf("focus locked: %s", map[bool]string{false: "none", true: "LogView"}[rootUI.focusLocked])
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
		if view.isVisible {
			obs.AddItem(view, 0, view.Proportion(), false)
			//obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.RemoveItem(view)
			//obs.ResizeItem(view, 0, 0)
		}
	}
}

func (view *MediaView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.AddItem(view, 0, view.Proportion(), false)
			//obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.RemoveItem(view)
			//obs.ResizeItem(view, 0, 0)
		}
	}
}

func (view *MediaDetailView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.AddItem(view, 0, view.Proportion(), false)
			//obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.RemoveItem(view)
			//obs.ResizeItem(view, 0, 0)
		}
	}
}

func (view *PlaylistView) SetVisible(visible bool) {
	infoLog.Log("resizing playlist")
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.AddItem(view, 0, view.Proportion(), false)
			//obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.RemoveItem(view)
			//obs.ResizeItem(view, 0, 0)
		}
	}
}

func (view *LogView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.AddItem(view, 0, view.Proportion(), false)
			//obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.RemoveItem(view)
			//obs.ResizeItem(view, 0, 0)
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
