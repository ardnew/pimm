package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	PopulateCycleDuration = 10 * time.Millisecond
)

type CellInfo struct {
	cell          *tview.TableCell
	library       *Library
	media         *Media
	libPath       string
	playlistIndex int
}

func NewCellInfo(cell *tview.TableCell, library *Library, media *Media) *CellInfo {
	return &CellInfo{
		cell:          cell,
		library:       library,
		media:         media,
		libPath:       fmt.Sprintf("%s//%s", library.Path(), media.Dir()),
		playlistIndex: PlaylistIndexInvalid,
	}
}

func (ref *CellInfo) String() string {
	return fmt.Sprintf("%s: %s", ref.library, ref.media)
}

type MediaView struct {
	*tview.Table
	sync.Mutex // coordinates visual updates
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	absolute   int
	isAbsolute bool
	isVisible  bool
	focusRune  rune
	mediaIndex []*CellInfo // table of all media
	tableIndex []int       // xref for current/unfiltered media
}

func NewMediaView(container *tview.Flex) *MediaView {

	mediaView := &MediaView{
		Table:      tview.NewTable(),
		Mutex:      sync.Mutex{},
		ui:         nil,
		obscura:    container,
		proportion: 3,
		absolute:   0,
		isAbsolute: false,
		isVisible:  true,
		focusRune:  MediaFocusRune,
		mediaIndex: []*CellInfo{},
		tableIndex: []int{},
	}
	container.SetTitle(" Media (m) ")
	container.SetBorder(true)
	mediaView.SetTitle("")
	mediaView.SetBorder(false)
	mediaView.SetBorderColor(tcell.ColorDarkSlateGray)
	mediaView.SetBorders(false)
	mediaView.SetSelectable(true /*rows*/, false /*cols*/)

	mediaView.SetSelectionChangedFunc(mediaView.mediaViewSelectionChanged)

	return mediaView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *MediaView) UI() *UI              { return view.ui.(*UI) }
func (view *MediaView) FocusRune() rune      { return view.focusRune }
func (view *MediaView) Obscura() *tview.Flex { return view.obscura }
func (view *MediaView) Proportion() int      { return view.proportion }
func (view *MediaView) Absolute() int        { return view.absolute }
func (view *MediaView) IsAbsolute() bool     { return view.isAbsolute }
func (view *MediaView) Visible() bool        { return view.isVisible }
func (view *MediaView) Resizable() bool      { return false }

func (view *MediaView) SetVisible(visible bool) {
	return
}

func (view *MediaView) LockFocus(lock bool) {

	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.Obscura().SetBorderColor(view.UI().focusLockedColor)
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
			//infoLog.Logf("table input: %v", event)
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.Table.InputHandler()(event, setFocus)
		})
}

// -----------------------------------------------------------------------------
//  (pimm) MediaView
// -----------------------------------------------------------------------------

func (view *MediaView) appendMedia(library *Library, media *Media) int {

	cell := tview.NewTableCell(media.Name())
	cell.SetExpansion(1)

	// multiple library scanners may be trying to add content to the table at
	// the same time. a mutex will ensure no table position conflicts
	view.Lock()
	row := view.GetRowCount()
	view.SetCell(row, 0, cell)
	view.mediaIndex = append(view.mediaIndex, NewCellInfo(cell, library, media))
	view.tableIndex = append(view.tableIndex, row)
	view.Unlock()

	return row
}

func (view *MediaView) AddMedia(library *Library, media *Media) {

	row := view.appendMedia(library, media)

	if nil == view.UI().mediaDetailView.media {
		view.mediaViewSelectionChanged(row, -1)
	}
}

func (view *MediaView) applyExcludeFilter(exclude map[string]byte) {

	view.tableIndex = []int{}
	view.Clear()

	count := 0
	for i, info := range view.mediaIndex {
		libPath := fmt.Sprintf("%s//%s", info.library.Path(), info.media.Dir())
		if _, excluded := exclude[libPath]; !excluded {
			view.tableIndex = append(view.tableIndex, i)
			view.SetCell(count, 0, info.cell)
			count++
			view.UI().SigDraw() <- view
		}
	}
}

func (view *MediaView) mediaViewSelectionChanged(row, column int) {

	info := view.mediaIndex[view.tableIndex[row]]

	view.UI().mediaDetailView.SetMedia(info.media)
}
