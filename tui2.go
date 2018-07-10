package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type MediaRef struct {
	library       *Library
	media         *Media
	libraryNode   *tview.TreeNode
	mediaCell     *tview.TableCell
	playlistIndex int
}

type LibraryView struct {
	*tview.TreeView
	container *tview.Flex
	isVisible bool
}

type MediaView struct {
	*tview.Table
	container *tview.Flex
	isVisible bool
}

type PlaylistView struct {
	*tview.List
	container *tview.Flex
	isVisible bool
}

type LogView struct {
	*tview.TextView
	container *tview.Flex
	isVisible bool
}

type UIView interface {
	FocusKey() tcell.Key
	SetVisible(bool)
	HandleKey(*tview.Application, *tcell.EventKey) *tcell.EventKey
}

type UI struct {
	app *tview.Application

	sigDraw chan interface{}

	media map[*Library]*MediaRef

	libraryView  *LibraryView
	mediaView    *MediaView
	playlistView *PlaylistView
	logView      *LogView
}

var (
	self            *tview.Application
	keyEventHandler = map[tcell.Key]UIView{}
)

func NewUI(opt *Options) *UI {

	self = tview.NewApplication()
	sigDraw := make(chan interface{})

	mediaRowsLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	browseLayout := tview.NewFlex().SetDirection(tview.FlexColumn)

	libraryView := &LibraryView{
		tview.NewTreeView(),
		browseLayout,
		true,
	}
	libraryView.SetTitle(" Library ")
	libraryView.SetBorder(true)
	libraryView.SetGraphics(true)
	libraryView.SetTopLevel(1)
	libraryView.SetInputCapture(nil)

	mediaView := &MediaView{
		tview.NewTable(),
		mediaRowsLayout,
		true,
	}
	mediaView.SetTitle(" Media ")
	mediaView.SetBorder(true)
	mediaView.SetBorders(false)
	mediaView.SetSelectable(true /*rows*/, false /*cols*/)
	mediaView.SetInputCapture(nil)

	playlistView := &PlaylistView{
		tview.NewList(),
		browseLayout,
		true,
	}
	playlistView.SetTitle(" Playlist ")
	playlistView.SetBorder(true)
	playlistView.ShowSecondaryText(false)
	playlistView.SetInputCapture(nil)

	logView := &LogView{
		tview.NewTextView(),
		mediaRowsLayout,
		true,
	}
	logView.SetTitle(" Log ")
	logView.SetBorder(true)
	logView.SetDynamicColors(true)
	logView.SetRegions(true)
	logView.SetScrollable(true)
	logView.SetWrap(false)
	logView.SetInputCapture(nil)

	mediaRowsLayout.AddItem(mediaView, 0, 3, false)
	mediaRowsLayout.AddItem(logView, 0, 1, false)

	browseLayout.AddItem(libraryView, 0, 2, false)
	browseLayout.AddItem(mediaRowsLayout, 0, 3, false)
	browseLayout.AddItem(playlistView, 0, 2, false)

	pageLayout := tview.NewPages()
	pageLayout.AddPage("browser", browseLayout, true, true)
	//pageLayout.AddPage("playlist", playlistView, true, false)

	self.SetRoot(pageLayout, true)
	self.SetFocus(mediaView)
	self.SetInputCapture(inputHandler)

	keyEventHandler[libraryView.FocusKey()] = libraryView
	keyEventHandler[mediaView.FocusKey()] = mediaView
	keyEventHandler[playlistView.FocusKey()] = playlistView
	keyEventHandler[tcell.KeyCtrlP] = playlistView
	keyEventHandler[logView.FocusKey()] = logView

	return &UI{
		app:     self,
		sigDraw: sigDraw,
		media:   make(map[*Library]*MediaRef),

		libraryView: libraryView,
		mediaView:   mediaView,
		logView:     logView,
	}
}

func inputHandler(event *tcell.EventKey) *tcell.EventKey {

	view, handled := keyEventHandler[event.Key()]
	if handled {
		return view.HandleKey(self, event)
	}

	return event
}

func (view *LibraryView) SetVisible(visible bool) {
	view.isVisible = visible
	if view.isVisible {
		view.container.AddItem(view, 0, 2, false)
	} else {
		view.container.RemoveItem(view)
	}
}

func (view *MediaView) SetVisible(visible bool) {
	view.isVisible = visible
	if view.isVisible {
		view.container.AddItem(view, 0, 3, false)
	} else {
		view.container.RemoveItem(view)
	}
}

func (view *PlaylistView) SetVisible(visible bool) {
	view.isVisible = visible
	if view.isVisible {
		view.container.AddItem(view, 0, 2, false)
	} else {
		view.container.RemoveItem(view)
	}
}

func (view *LogView) SetVisible(visible bool) {
	view.isVisible = visible
	if view.isVisible {
		view.container.AddItem(view, 0, 1, false)
	} else {
		view.container.RemoveItem(view)
	}
}

func (*LibraryView) FocusKey() tcell.Key  { return tcell.KeyF9 }
func (*MediaView) FocusKey() tcell.Key    { return tcell.KeyF10 }
func (*PlaylistView) FocusKey() tcell.Key { return tcell.KeyF11 }
func (*LogView) FocusKey() tcell.Key      { return tcell.KeyF12 }

func (view *LibraryView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch key {
	case view.FocusKey():
		prevFocus := app.GetFocus()
		if view.TreeView != prevFocus {
			if nil != prevFocus {
				prevFocus.Blur()
			}
			app.SetFocus(view)
		}
	}
	return event
}

func (view *MediaView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch key {
	case view.FocusKey():
		prevFocus := app.GetFocus()
		if view.Table != prevFocus {
			if nil != prevFocus {
				prevFocus.Blur()
			}
			app.SetFocus(view)
		}
	}
	return event
}

func (view *PlaylistView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch key {
	case view.FocusKey():
		prevFocus := app.GetFocus()
		if view.List != prevFocus {
			if nil != prevFocus {
				prevFocus.Blur()
			}
			app.SetFocus(view)
		}
	case tcell.KeyCtrlP:
		view.SetVisible(!view.isVisible)
	}
	return event
}

func (view *LogView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch key {
	case view.FocusKey():
		prevFocus := app.GetFocus()
		if view.TextView != prevFocus {
			if nil != prevFocus {
				prevFocus.Blur()
			}
			app.SetFocus(view)
		}
	}
	return event
}
