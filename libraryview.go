package main

import (
	"os"
	"path"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type NodeInfo struct {
	parent *tview.TreeNode
	path   string
}

type LibraryView struct {
	*tview.TreeView
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
	dirIndex   map[*Library]map[string]*tview.TreeNode
}

func NewLibraryView(container *tview.Flex) *LibraryView {

	libraryView := &LibraryView{
		TreeView:   tview.NewTreeView(),
		ui:         nil,
		obscura:    container,
		proportion: 1,
		isVisible:  true,
		focusRune:  LibraryFocusRune,
		dirIndex:   make(map[*Library]map[string]*tview.TreeNode),
	}
	libraryView.SetTitle(" Library (l) ")
	libraryView.SetBorder(true)
	libraryView.SetGraphics(true)
	libraryView.SetTopLevel(0)
	libraryView.SetSelectedFunc(nodeSelected)
	libraryView.SetChangedFunc(nil)

	host, err := os.Hostname()
	if err != nil {
		warnLog.Logf("failed to get hostname: %s", err)
		host = "localhost"
	}

	root := tview.NewTreeNode(host).SetSelectable(false)
	libraryView.SetRoot(root)
	libraryView.SetCurrentNode(root)

	return libraryView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *LibraryView) UI() *UI              { return view.ui.(*UI) }
func (view *LibraryView) FocusRune() rune      { return view.focusRune }
func (view *LibraryView) Obscura() *tview.Flex { return view.obscura }
func (view *LibraryView) Proportion() int      { return view.proportion }
func (view *LibraryView) Visible() bool        { return view.isVisible }
func (view *LibraryView) Resizable() bool      { return false }

func (view *LibraryView) SetVisible(visible bool) {

	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if visible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			//if view.ui.pageControl.focusedView == view {
			//	view.LockFocus(false)
			//}
		}
	}
}

func (view *LibraryView) LockFocus(lock bool) {

	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

// -----------------------------------------------------------------------------
//  (tview) embedded Primitive.(TreeView)
// -----------------------------------------------------------------------------

func (view *LibraryView) Focus(delegate func(p tview.Primitive)) {

	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[true])
		view.SetBorderColor(view.UI().focusBorderColor[true])
	}
	view.TreeView.Focus(delegate)
}

func (view *LibraryView) Blur() {

	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[false])
		view.SetBorderColor(view.UI().focusBorderColor[false])
	}
	view.TreeView.Blur()
}

func (view *LibraryView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}

			inputHandled := false

			currNode := view.GetCurrentNode()
			currInfo := currNode.GetReference().(*NodeInfo)

			switch mask := event.Modifiers(); mask {

			case tcell.ModAlt:

				switch event.Key() {
				case tcell.KeyLeft:
					for n := currNode; nil != n; {
						n.Collapse()
						n = n.GetReference().(*NodeInfo).parent
					}
					inputHandled = true
				case tcell.KeyRight:
					currNode.ExpandAll()
					inputHandled = true
				}

			case tcell.ModNone:

				switch event.Key() {
				case tcell.KeyLeft:
					children := currNode.GetChildren()
					isExpanded := currNode.IsExpanded() && len(children) > 0
					if nil != currInfo.parent && !isExpanded {
						view.SetCurrentNode(currInfo.parent)
						currInfo.parent.Collapse()
					} else {
						currNode.Collapse()
					}
					inputHandled = true
				case tcell.KeyRight:
					currNode.Expand()
				}
			}

			if !inputHandled {
				view.TreeView.InputHandler()(event, setFocus)
			}
		})
}

// -----------------------------------------------------------------------------
//  (pimm) LibraryView
// -----------------------------------------------------------------------------

func nodeSelected(node *tview.TreeNode) {

	node.SetExpanded(!node.IsExpanded())
}

func (view *LibraryView) AddLibrary(library *Library) {

	node := tview.NewTreeNode(library.Name())
	node.SetSelectable(true)
	node.Collapse()
	node.SetReference(&NodeInfo{nil, library.Path()})

	root := view.GetRoot()
	root.AddChild(node)

	view.dirIndex[library] = make(map[string]*tview.TreeNode)
	view.dirIndex[library][library.Path()] = node
}

func (view *LibraryView) AddLibraryDirectory(library *Library, dir string) {

	libIndex := view.dirIndex[library]
	parent := libIndex[path.Dir(dir)]
	node := tview.NewTreeNode(path.Base(dir))

	parent.AddChild(node)
	node.Collapse()
	node.SetReference(&NodeInfo{parent, dir})

	libIndex[dir] = node
}
