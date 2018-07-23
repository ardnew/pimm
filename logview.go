package main

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type LogView struct {
	*tview.TextView
	ui         interface{}
	obscura    *tview.Flex
	proportion int
	isVisible  bool
	focusRune  rune
}

func NewLogView(container *tview.Flex) *LogView {

	logView := &LogView{
		TextView:   tview.NewTextView(),
		ui:         nil,
		obscura:    container,
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

	return logView
}

// -----------------------------------------------------------------------------
//  (pimm) UIView interface
// -----------------------------------------------------------------------------

func (view *LogView) UI() *UI              { return view.ui.(*UI) }
func (view *LogView) FocusRune() rune      { return view.focusRune }
func (view *LogView) Obscura() *tview.Flex { return view.obscura }
func (view *LogView) Proportion() int      { return view.proportion }
func (view *LogView) Visible() bool        { return view.isVisible }
func (view *LogView) Resizable() bool      { return true }

func (view *LogView) SetVisible(visible bool) {
	view.isVisible = visible
	obs := view.Obscura()
	if nil != obs {
		if view.isVisible {
			obs.ResizeItem(view, 0, view.Proportion())
		} else {
			obs.ResizeItem(view, 2, 0)
			if view.UI().pageControl.focusedView == view {
				view.LockFocus(false)
			}
		}
	}
}

func (view *LogView) LockFocus(lock bool) {
	view.UI().focusLocked = lock
	view.UI().focusLockedView = view
	if lock {
		view.SetBorderColor(tcell.ColorDodgerBlue)
	} else {
		view.SetBorderColor(view.UI().focusBorderColor[view.UI().pageControl.focusedView == view])
	}
}

// -----------------------------------------------------------------------------
//  (tview) embedded Primitive.(TextView)
// -----------------------------------------------------------------------------

func (view *LogView) Focus(delegate func(p tview.Primitive)) {
	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[true])
		view.SetBorderColor(view.UI().focusBorderColor[true])
	}
	view.TextView.Focus(delegate)
}

func (view *LogView) Blur() {
	if nil != view.ui {
		view.SetTitleColor(view.UI().focusTitleColor[false])
		view.SetBorderColor(view.UI().focusBorderColor[false])
	}
	view.TextView.Blur()
}

func (view *LogView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return view.WrapInputHandler(
		func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			if view.UI().GlobalInputHandled(view, event, setFocus) {
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

// -----------------------------------------------------------------------------
//  (pimm) LogView
// -----------------------------------------------------------------------------
