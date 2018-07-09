package main

import (
	"fmt"
	"path"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type LibSubdirKey struct {
	layout  *Layout
	path    string
	library *Library
}

func (key *LibSubdirKey) String() string {
	return fmt.Sprintf("%s|%s", key.path, key.library)
}

type Layout struct {
	app         *tview.Application
	options     *Options
	updateQueue *chan bool
	mediaIndex  *map[string]*tview.TableCell
	libIndex    *map[string]*tview.TreeNode
	libTree     *tview.TreeView
	mediaTable  *tview.Table
}

const (
	TreeNodeColorDir      = tcell.ColorGreen
	TreeNodeColorDirEmpty = tcell.ColorWhite
)

const (
	DrawUpdateDuration = 100 * time.Millisecond
	DrawCycleDuration  = 10 * DrawUpdateDuration
)

// creates and initializes the tview primitives that will constitute the layout
// of our primary view
func initUI(opt *Options) *Layout {

	// the root tview object "app" coordinates all of the tview primitives with
	// the tcell screen object, who in-turn coordinates that screen object with
	// our actual terminal
	app := tview.NewApplication()

	// we create channel "updateQueue" to be polled periodically[1] by a lone
	// wolf goroutine.
	//   NOTE[1]: periodically means polled every "DrawUpdateDuration" (above)
	updateQueue := make(chan bool)
	setLogUpdate(&updateQueue)

	// decided to use a tview.Table for displaying the most important view of
	// this application -- the library media browser
	mediaIndex := make(map[string]*tview.TableCell)
	mediaTable := tview.NewTable()
	mediaTable.SetBorder(true)   // whole table
	mediaTable.SetBorders(false) // inbetween cells
	mediaTable.SetSelectable(true /*rows*/, false /*cols*/)
	mediaTable.SetTitle(" Media ")

	// created a tview.TreeView to present the libraries and the subdirs in a
	// collapsable tree data structure mirroring a fs tree structure
	libIndex := make(map[string]*tview.TreeNode)
	libTree := tview.NewTreeView()
	rootNode := tview.NewTreeNode("<ROOT>").SetColor(tcell.ColorRed).SetSelectable(false)
	libTree.SetRoot(rootNode).SetCurrentNode(rootNode)
	libTree.SetBorder(true).SetTitle(" Library ")
	libTree.SetGraphics(true)
	libTree.SetTopLevel(1)
	libTree.SetSelectedFunc(libraryNodeSelected)
	libTree.SetChangedFunc(libraryNodeChanged)

	logView := tview.NewTextView()
	logView.SetTitle(" Log ")
	logView.SetBorder(true)
	logView.SetDynamicColors(true)
	logView.SetRegions(true)
	logView.SetScrollable(true)
	logView.SetWrap(false)
	setLogWriter(logView)

	browseLayout := tview.NewFlex().SetDirection(tview.FlexColumn)
	browseLayout = browseLayout.AddItem(libTree, 0, 1, false)
	browseLayout = browseLayout.AddItem(mediaTable, 0, 3, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mainLayout = mainLayout.AddItem(browseLayout, 0, 3, false)
	mainLayout = mainLayout.AddItem(logView, 0, 1, false)

	app.SetRoot(mainLayout, true).SetFocus(libTree)

	layout := &Layout{
		app:         app,
		options:     opt,
		updateQueue: &updateQueue,
		mediaIndex:  &mediaIndex,
		libIndex:    &libIndex,
		libTree:     libTree,
		mediaTable:  mediaTable,
	}

	go layout.DrawCycle()

	return layout
}

// spawn a goroutine that will update the UI only whenever we have a change
// made to one of the subviews, as notified via the bool channel "draw"
func (y *Layout) DrawCycle() {
	// multiple goroutines may post to the channel in a very short timespan.
	// so instead of redrawing every time, notify the channel that an update
	// is available, Draw() once, and then drain the queue
	cycle := time.Tick(DrawCycleDuration)
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
		case <-cycle:
			// also perform unconditional draw updates at a regular interval to
			// theoretically account for any changes i accidentally miss making
			// an explicit request for -- on the incredibly low chance i mess up
			y.app.Draw()
		default:
		}
	}
}

func updatePrefix(node *tview.TreeNode) {

	switch ref := node.GetReference().(type) {
	case *LibSubdirKey:
		text := path.Base(ref.path)
		layout := ref.layout
		isUTF8, isExpanded := layout.options.UTF8.bool, node.IsExpanded()
		prefix := treeNodePrefixExpanded[isUTF8][isExpanded]
		node.SetText(fmt.Sprintf("%s%s", prefix, text))
	}
}

func libraryNodeSelected(node *tview.TreeNode) {

	node.SetExpanded(!node.IsExpanded())
	updatePrefix(node)
}

func libraryNodeChanged(node *tview.TreeNode) {

	switch ref := node.GetReference().(type) {
	case *LibSubdirKey:
		ref.layout.ShowMedia(ref)
	}
}

func (y *Layout) AddLibraryNode(lib *Library, path, name, parent string) {

	var (
		parentNode *tview.TreeNode
		nodeColor  tcell.Color
		nodeText   string
	)

	isRoot := len(parent) == 0

	parentKey := &LibSubdirKey{y, parent, lib}
	key := &LibSubdirKey{y, path, lib}

	index := *y.libIndex
	if isRoot {
		parentNode = y.libTree.GetRoot()
		nodeColor = TreeNodeColorDir
		nodeText = fmt.Sprintf("%s", name)
	} else {
		parentNode = index[parentKey.String()]
		nodeColor = TreeNodeColorDirEmpty
		nodeText = fmt.Sprintf("%s", name)
	}

	node := tview.NewTreeNode(nodeText)
	node.SetColor(nodeColor)
	node.SetSelectable(true)
	node.SetExpanded(isRoot)
	node.SetReference(key)
	updatePrefix(node)

	parentNode.AddChild(node)
	index[key.String()] = node

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

func (y *Layout) ShowMedia(key *LibSubdirKey) {

	y.mediaTable = y.mediaTable.Clear()

	table := key.library.Media()
	media, ok := table[key.path]

	if !ok {
		warnLog.Logf("failed to read media file index:\n\t%q", key.path)
	}

	for _, m := range media {
		y.AddMediaTableRow(key.library, m)
	}
}
