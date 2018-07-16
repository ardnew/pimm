package main

import (
	"path"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

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

	root := tview.NewTreeNode("Library").SetSelectable(false)
	libraryView.SetRoot(root)
	libraryView.SetCurrentNode(root)

	return libraryView
}

func (view *LibraryView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
				return
			}
			view.TreeView.InputHandler()(event, setFocus)
		})
}

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

func (view *LibraryView) UI() *UI              { return view.ui.(*UI) }
func (view *LibraryView) FocusRune() rune      { return view.focusRune }
func (view *LibraryView) Obscura() *tview.Flex { return view.obscura }
func (view *LibraryView) Proportion() int      { return view.proportion }

func (view *LibraryView) LockFocus(lock bool) {
	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

func (view *LibraryView) Visible() bool { return view.isVisible }
func (view *LibraryView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if visible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.UI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func nodeSelected(node *tview.TreeNode) {

	node.SetExpanded(!node.IsExpanded())
}

func (view *LibraryView) AddLibrary(library *Library) {

	node := tview.NewTreeNode(library.Name())
	node.SetSelectable(true)
	node.SetExpanded(true)

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
	node.SetExpanded(true)

	libIndex[dir] = node
}

func (view *LibraryView) AddMedia(library *Library, media *Media) {
	infoLog.Logf("discovered: %s", media)
}
