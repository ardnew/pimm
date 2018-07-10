package main

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type LibDirKey struct {
	layout  *Layout
	path    string
	library *Library
}

func (key *LibDirKey) String() string {
	return fmt.Sprintf("%s|%s", key.path, key.library)
}

func (key *LibDirKey) Node() *tview.TreeNode {
	index := *key.layout.libIndex
	if node, ok := index[key.String()]; ok {
		return node
	}
	return nil
}

type PimmView interface {
	FocusKey() tcell.Key
	HandleKey(*tview.Application, *tcell.EventKey) *tcell.EventKey
}

type PVTreeView struct{ *tview.TreeView }
type PVTableView struct{ *tview.Table }
type PVTextView struct{ *tview.TextView }

type Layout struct {
	app         *tview.Application
	options     *Options
	updateQueue *chan bool
	selectedKey *LibDirKey
	mediaIndex  *map[string]*tview.TableCell
	libIndex    *map[string]*tview.TreeNode
	libTree     *PVTreeView
	mediaTable  *PVTableView
	logView     *PVTextView
}

const (
	TreeNodeColorSelected    = tcell.ColorGreen
	TreeNodeColorNotSelected = tcell.ColorWhite
)

const (
	DrawUpdateDuration = 100 * time.Millisecond
	DrawCycleDuration  = 10 * DrawUpdateDuration
)

var (
	app        *tview.Application
	keyHandler = map[tcell.Key]PimmView{}
)

func appInputHandler(event *tcell.EventKey) *tcell.EventKey {

	// if appInputHandler was called, then it was succesfully installed on the
	// unit-visible "app" and is thus non-nil
	view, handle := keyHandler[event.Key()]
	if handle {
		return view.HandleKey(app, event)
	}

	return event
}

func (view *PVTreeView) FocusKey() tcell.Key {
	return tcell.KeyF1
}

func (view *PVTreeView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {

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

	case tcell.KeyF5:
		node := view.GetCurrentNode()
		switch ref := node.GetReference().(type) {
		case *LibDirKey:
			go func(r *LibDirKey) {
				r.layout.ShowMedia(r)
			}(ref)
		}
	}
	return event
}

func (view *PVTableView) FocusKey() tcell.Key {
	return tcell.KeyF2
}

func (view *PVTableView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {

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

func (view *PVTextView) FocusKey() tcell.Key {
	return tcell.KeyF12
}

func (view *PVTextView) HandleKey(app *tview.Application, event *tcell.EventKey) *tcell.EventKey {

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

// creates and initializes the tview primitives that will constitute the layout
// of our primary view
func initUI(opt *Options) *Layout {

	// the root tview object "app" coordinates all of the tview primitives with
	// the tcell screen object, who in-turn coordinates that screen object with
	// our actual terminal
	app = tview.NewApplication()
	app.SetInputCapture(appInputHandler)

	// we create channel "updateQueue" to be polled periodically[1] by a lone
	// wolf goroutine.
	//   NOTE[1]: periodically means polled every "DrawUpdateDuration" (above)
	updateQueue := make(chan bool)
	//setLogUpdate(&updateQueue)

	// decided to use a tview.Table for displaying the most important view of
	// this application -- the library media browser
	mediaIndex := make(map[string]*tview.TableCell)
	mediaTable := &PVTableView{tview.NewTable()}
	mediaTable.SetBorder(true)   // whole table
	mediaTable.SetBorders(false) // inbetween cells
	mediaTable.SetBackgroundColor(tcell.ColorBlack)
	mediaTable.SetSelectable(true /*rows*/, false /*cols*/)
	mediaTable.SetTitle(" Media ")
	mediaTable.SetInputCapture(nil)

	// created a tview.TreeView to present the libraries and the subdirs in a
	// collapsable tree data structure mirroring a fs tree structure
	libIndex := make(map[string]*tview.TreeNode)
	libTree := &PVTreeView{tview.NewTreeView()}
	rootNode := tview.NewTreeNode("<ROOT>")
	libTree.SetRoot(rootNode).SetCurrentNode(rootNode)
	libTree.SetBorder(true).SetTitle(" Library ")
	libTree.SetGraphics(true)
	libTree.SetTopLevel(1)
	libTree.SetSelectedFunc(libTreeNodeSelected)
	libTree.SetChangedFunc(libTreeNodeChanged)
	libTree.SetInputCapture(nil)

	logView := &PVTextView{tview.NewTextView()}
	logView.SetTitle(" Log ")
	logView.SetBorder(true)
	logView.SetDynamicColors(true)
	logView.SetRegions(true)
	logView.SetScrollable(true)
	logView.SetWrap(false)
	logView.SetInputCapture(nil)
	setLogWriter(logView)

	browseLayout := tview.NewFlex().SetDirection(tview.FlexColumn)
	browseLayout.AddItem(libTree, 0, 1, false)
	browseLayout.AddItem(mediaTable, 0, 3, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mainLayout.AddItem(browseLayout, 0, 3, false)
	mainLayout.AddItem(logView, 0, 1, false)

	app.SetRoot(mainLayout, true).SetFocus(libTree)

	keyHandler[libTree.FocusKey()] = libTree
	keyHandler[tcell.KeyF5] = libTree
	keyHandler[mediaTable.FocusKey()] = mediaTable
	keyHandler[logView.FocusKey()] = logView

	layout := &Layout{
		app:         app,
		options:     opt,
		updateQueue: &updateQueue,
		mediaIndex:  &mediaIndex,
		libIndex:    &libIndex,
		libTree:     libTree,
		mediaTable:  mediaTable,
		logView:     logView,
	}

	//	rootKey := &LibDirKey{layout, "/", nil}
	//	rootNode.SetColor(TreeNodeColorNotSelected)
	//	rootNode.SetSelectable(true)
	//	rootNode.SetExpanded(true)
	//	rootNode.SetReference(rootKey)
	//	updatePrefix(rootNode)
	//	libIndex[rootKey.String()] = rootNode

	go layout.DrawCycle()

	return layout
}

// spawn a goroutine that will update the UI only whenever we have a change
// made to one of the subviews, as notified via the bool channel "draw"
func (y *Layout) DrawCycle() {
	// multiple goroutines may post to the channel in a very short timespan.
	// so instead of redrawing every time, notify the channel that an update
	// is available, Draw() once, and then drain the queue
	//cycle := time.Tick(DrawCycleDuration)
	update := time.Tick(DrawUpdateDuration)
	for {
		select {
		case <-update:
			// sufficient time has elapsed
			select {
			case <-*y.updateQueue:
				// sufficient time elapsed AND we have an update available
				y.app.Draw()
				for empty := false; !empty; {
					select {
					case <-*y.updateQueue:
					default:
						empty = true
					}
				}
			default:
			}
		//case <-cycle:
		// also perform unconditional draw updates at a regular interval to
		// theoretically account for any changes i accidentally miss making
		// an explicit request for -- on the incredibly low chance i mess up
		//y.app.Draw()
		default:
		}
	}
}

func updatePrefix(node *tview.TreeNode) {

	switch ref := node.GetReference().(type) {
	case *LibDirKey:
		text := path.Base(ref.path)
		layout := ref.layout
		isUTF8, isExpanded := layout.options.UTF8.bool, node.IsExpanded()
		prefix := treeNodePrefixExpanded[isExpanded][isUTF8][false]
		node.SetText(fmt.Sprintf("%s%s", prefix, text))
		*layout.updateQueue <- true
	}
}

func libTreeNodeSelected(node *tview.TreeNode) {

	node.SetExpanded(!node.IsExpanded())
	updatePrefix(node)
}

func libTreeNodeChanged(node *tview.TreeNode) {

}

func (y *Layout) AddLibraryNode(lib *Library, path, name, parent string) {

	var parentNode *tview.TreeNode

	isRoot := len(parent) == 0

	if isRoot {
		parentNode = y.libTree.GetRoot()
	} else {
		parentKey := &LibDirKey{y, parent, lib}
		parentNode = parentKey.Node()
	}

	key := &LibDirKey{y, path, lib}
	nodeText := fmt.Sprintf("%s", name)
	node := tview.NewTreeNode(nodeText)
	node.SetColor(TreeNodeColorNotSelected)
	node.SetSelectable(true)
	node.SetExpanded(isRoot)
	node.SetReference(key)
	updatePrefix(node)

	index := *y.libIndex
	index[key.String()] = node
	parentNode.AddChild(node)

	*y.updateQueue <- true
}

func (y *Layout) AddLibrary(lib *Library) {
	y.AddLibraryNode(lib, lib.Path(), lib.Name(), "" /* root paths have no parent */)
}

func (y *Layout) AddLibrarySubdir(lib *Library, subdir string) {
	y.AddLibraryNode(lib, subdir, path.Base(subdir), path.Dir(subdir))
}

func (y *Layout) AddMediaTableRow(lib *Library, media *Media) {

	var cell *tview.TableCell

	row := y.mediaTable.GetRowCount()
	for col, val := range media.Columns() {
		cell = tview.NewTableCell(val)
		cell.SetAlign(tview.AlignLeft)
		y.mediaTable.SetCell(row, col, cell)
	}

	index := *y.mediaIndex
	mediaPath := media.Path()
	_, exists := index[mediaPath]

	if !exists {
		index[mediaPath] = y.mediaTable.GetCell(0, 0)
	}
	*y.updateQueue <- true
}

func (y *Layout) AddMedia(lib *Library, media *Media) {

	index := *y.mediaIndex
	mediaPath := media.Path()
	_, exists := index[mediaPath]

	// one of the other libraries may have already found this file and is
	// currntly displaying it in the media browser. we dont want duplicates
	// --> so keep it simple and only show the file one time
	if !exists {
		y.AddMediaTableRow(lib, media)
	}
}

func (y *Layout) ShowMedia(key *LibDirKey) {

	y.mediaTable.Clear()

	// unselect previous node
	if prevKey := y.selectedKey; nil != prevKey {
		if node := prevKey.Node(); nil != node {
			node.SetColor(TreeNodeColorNotSelected)
		}
	}

	// select current node
	if node := key.Node(); nil != node {
		y.selectedKey = key
		node.SetColor(TreeNodeColorSelected)
	}

	table := key.library.Media()

	infoLog.Logf("refreshing library media: %q", key.path)
	for path, content := range table {
		if strings.HasPrefix(path, key.path) {
			for _, m := range content {
				y.AddMediaTableRow(key.library, m)
			}
		}
	}
}
