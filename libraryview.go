package main

import (
	"fmt"
	"os"
	"path"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type NodeInfo struct {
	library *Library
	parent  *tview.TreeNode
	path    string
	include bool
}

type LibraryView struct {
	*tview.TreeView
	ui           interface{}
	obscura      *tview.Flex
	proportion   int
	absolute     int
	isAbsolute   bool
	isVisible    bool
	focusRune    rune
	selectedNode *tview.TreeNode
	dirIndex     map[*Library]map[string]*tview.TreeNode
}

func NewLibraryView(container *tview.Flex) *LibraryView {

	libraryView := &LibraryView{
		TreeView:   tview.NewTreeView(),
		ui:         nil,
		obscura:    container,
		proportion: 1,
		absolute:   0,
		isAbsolute: false,
		isVisible:  true,
		focusRune:  LibraryFocusRune,
		dirIndex:   make(map[*Library]map[string]*tview.TreeNode),
	}
	libraryView.SetTitle(" Library (l) ")
	libraryView.SetBorder(true)
	libraryView.SetGraphics(true)
	libraryView.SetTopLevel(0)
	libraryView.SetSelectedFunc(libraryView.nodeSelected)
	libraryView.SetChangedFunc(libraryView.nodeChanged)

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
func (view *LibraryView) Absolute() int        { return view.absolute }
func (view *LibraryView) IsAbsolute() bool     { return view.isAbsolute }
func (view *LibraryView) Visible() bool        { return view.isVisible }
func (view *LibraryView) Resizable() bool      { return false }

func (view *LibraryView) SetVisible(visible bool) {

	return
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
						view.expandNode(n, false)
						n = n.GetReference().(*NodeInfo).parent
					}
					inputHandled = true
				}

			case tcell.ModNone:

				switch event.Key() {
				case tcell.KeyLeft:
					view.expandNode(currNode, false)
					if nil != currInfo.parent {
						view.SetCurrentNode(currInfo.parent)
						view.nodeChanged(currInfo.parent)
						inputHandled = true
					}
				case tcell.KeyRight:
					view.expandNode(currNode, true)
					// be sure to continue on with TreeView's input handler
					inputHandled = false

				case tcell.KeyRune:
					switch event.Rune() {
					case '[':
						view.includeNode(currNode, true)
						inputHandled = true
					case ']':
						view.includeNode(currNode, false)
						inputHandled = true
					}
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

func (view *LibraryView) updateNodeAppearance(node *tview.TreeNode, isSelected bool) {

	info := node.GetReference().(*NodeInfo)
	name := path.Base(info.path)

	isColor := true
	isUnicode := view.UI().options.UTF8.bool
	isExpanded := node.IsExpanded()
	isIncluded := info.include

	pfix := treeNodePrefixExpanded[isExpanded][isUnicode][isColor]
	incl := treeNodePrefixIncluded[isIncluded][isUnicode][isColor]

	if isSelected {
		incl = "[black:white]"
	}

	node.SetText(fmt.Sprintf("%s%s%s", pfix, incl, name))
}

func (view *LibraryView) expandNode(node *tview.TreeNode, expand bool) {

	isSelectedNode := node == view.selectedNode

	node.SetExpanded(expand)

	view.updateNodeAppearance(node, isSelectedNode)
}

func (view *LibraryView) includeNode(node *tview.TreeNode, include bool) {

	isSelectedNode := node == view.selectedNode

	info := node.GetReference().(*NodeInfo)
	info.include = include

	view.updateNodeAppearance(node, isSelectedNode)
}

func (view *LibraryView) nodeSelected(node *tview.TreeNode) {

	// on selection, simply toggle the node's expansion state
	view.expandNode(node, !node.IsExpanded())
}

func (view *LibraryView) nodeChanged(node *tview.TreeNode) {

	if nil != view.selectedNode {
		view.updateNodeAppearance(view.selectedNode, false)
	}
	view.selectedNode = node
	view.updateNodeAppearance(node, true)
}

func (view *LibraryView) AddLibrary(library *Library) {

	node := tview.NewTreeNode(library.Name())
	node.SetSelectable(true)
	node.SetReference(&NodeInfo{library, nil, library.Path(), true})

	root := view.GetRoot()
	root.AddChild(node)

	if 1 == len(root.GetChildren()) {
		view.nodeChanged(node)
	}

	//infoLog.Logf("sel = %#v", view.selectedNode)
	//infoLog.Logf("sel = %#v", root)
	//infoLog.Logf("sel = %#v", node)

	view.dirIndex[library] = make(map[string]*tview.TreeNode)
	view.dirIndex[library][library.Path()] = node

	view.expandNode(node, false)
}

func (view *LibraryView) AddLibraryDirectory(library *Library, dir string) {

	libIndex := view.dirIndex[library]
	parent := libIndex[path.Dir(dir)]
	node := tview.NewTreeNode(path.Base(dir))

	parent.AddChild(node)
	node.SetReference(&NodeInfo{library, parent, dir, true})

	libIndex[dir] = node

	view.expandNode(node, false)
}
