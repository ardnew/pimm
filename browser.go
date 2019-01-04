// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 01 Jan 2019
//  FILE: browser.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    based on tview.List, this implements all methods of the tview.Primitive
//    interface. this affords an easier means of changing the appearance of the
//    tview.List widget used for presenting media lists to the user.
//
// =============================================================================

package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// mediaItem represents one item in a Browser.
type mediaItem struct {
	MainText      string // The main text of the list item.
	SecondaryText string // A secondary text to be shown underneath the main text.
	Selected      func() // The optional function which is called when the item is selected.
}

// Browser displays rows of items, each of which can be selected.
type Browser struct {
	*tview.Box

	// The items of the list.
	items []*mediaItem

	// The index of the currently selected item.
	currentItem int

	// The offset to ensure our currently selected item remains in view.
	viewOffset int

	// Whether or not to show the secondary item texts.
	showSecondaryText bool

	// The item main text color.
	mainTextColor tcell.Color

	// The item secondary text color.
	secondaryTextColor tcell.Color

	// The text color for selected items.
	selectedTextColor tcell.Color

	// The background color for selected items.
	selectedBackgroundColor tcell.Color

	// If true, the selection is only shown when the list has focus.
	selectedFocusOnly bool

	// An optional function which is called when the user has navigated to a list
	// item.
	changed func(index int, mainText, secondaryText string)

	// An optional function which is called when a list item was selected. This
	// function will be called even if the list item defines its own callback.
	selected func(index int, mainText, secondaryText string)

	// An optional function which is called when the user presses the Escape key.
	done func()
}

// NewBrowser returns a new form.
func NewBrowser() *Browser {
	return &Browser{
		Box:                     tview.NewBox(),
		showSecondaryText:       true,
		mainTextColor:           colorScheme.activeText,
		secondaryTextColor:      colorScheme.inactiveText,
		selectedTextColor:       colorScheme.backgroundPrimary,
		selectedBackgroundColor: colorScheme.highlightPrimary,
	}
}

// SetCurrentItem sets the currently selected item by its index. This triggers
// a "changed" event.
func (l *Browser) SetCurrentItem(index int) *Browser {
	l.currentItem = index
	if l.currentItem < len(l.items) && l.changed != nil {
		item := l.items[l.currentItem]
		l.changed(l.currentItem, item.MainText, item.SecondaryText)
	}
	return l
}

// GetCurrentItem returns the index of the currently selected list item.
func (l *Browser) GetCurrentItem() int {
	return l.currentItem
}

// RemoveItem removes the item with the given index (starting at 0) from the
// list. Does nothing if the index is out of range.
func (l *Browser) RemoveItem(index int) *Browser {
	if index < 0 || index >= len(l.items) {
		return l
	}
	l.items = append(l.items[:index], l.items[index+1:]...)
	if l.currentItem >= len(l.items) {
		l.currentItem = len(l.items) - 1
	}
	// TODO:
	//   maybe decrement currentItem if it equals the removal index so that it
	//   shifts down along with the subsequent items? which behavior feels more
	//   natural, intuitive?
	if l.currentItem < len(l.items) && l.changed != nil {
		item := l.items[l.currentItem]
		l.changed(l.currentItem, item.MainText, item.SecondaryText)
	}
	return l
}

// SetMainTextColor sets the color of the items' main text.
func (l *Browser) SetMainTextColor(color tcell.Color) *Browser {
	l.mainTextColor = color
	return l
}

// SetSecondaryTextColor sets the color of the items' secondary text.
func (l *Browser) SetSecondaryTextColor(color tcell.Color) *Browser {
	l.secondaryTextColor = color
	return l
}

// SetSelectedTextColor sets the text color of selected items.
func (l *Browser) SetSelectedTextColor(color tcell.Color) *Browser {
	l.selectedTextColor = color
	return l
}

// SetSelectedBackgroundColor sets the background color of selected items.
func (l *Browser) SetSelectedBackgroundColor(color tcell.Color) *Browser {
	l.selectedBackgroundColor = color
	return l
}

// SetSelectedFocusOnly sets a flag which determines when the currently selected
// list item is highlighted. If set to true, selected items are only highlighted
// when the list has focus. If set to false, they are always highlighted.
func (l *Browser) SetSelectedFocusOnly(focusOnly bool) *Browser {
	l.selectedFocusOnly = focusOnly
	return l
}

// ShowSecondaryText determines whether or not to show secondary item texts.
func (l *Browser) ShowSecondaryText(show bool) *Browser {
	l.showSecondaryText = show
	return l
}

// SetChangedFunc sets the function which is called when the user navigates to
// a list item. The function receives the item's index in the list of items
// (starting with 0), its main text, and its secondary text.
//
// This function is also called when the first item is added or when
// SetCurrentItem() is called.
func (l *Browser) SetChangedFunc(handler func(index int, mainText string, secondaryText string)) *Browser {
	l.changed = handler
	return l
}

// SetSelectedFunc sets the function which is called when the user selects a
// list item by pressing Enter on the current selection. The function receives
// the item's index in the list of items (starting with 0), its main text, and
// its secondary text.
func (l *Browser) SetSelectedFunc(handler func(int, string, string)) *Browser {
	l.selected = handler
	return l
}

// SetDoneFunc sets a function which is called when the user presses the Escape
// key.
func (l *Browser) SetDoneFunc(handler func()) *Browser {
	l.done = handler
	return l
}

// AddItem adds a new item to the list. An item has a main text which will be
// highlighted when selected. It also has a secondary text which is shown
// underneath the main text (if it is set to visible) but which may remain
// empty.
//
// The "selected" callback will be invoked when the user selects the item. You
// may provide nil if no such item is needed or if all events are handled
// through the selected callback set with SetSelectedFunc().
func (l *Browser) AddItem(mainText, secondaryText string, selected func()) *Browser {
	l.items = append(l.items, &mediaItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Selected:      selected,
	})
	if len(l.items) == 1 && l.changed != nil {
		item := l.items[0]
		l.changed(0, item.MainText, item.SecondaryText)
	}
	return l
}

func (l *Browser) InsertItem(index int, mainText, secondaryText string, selected func()) *Browser {

	// several different ways to interpret index < 0. one convenient way would
	// be to insert starting from the end of the list. the safest option, which
	// is implemented here, is to just consider it as invalid input and return
	// the original list unmodified.
	if index < 0 {
		return l
	}

	// if the index provided is greater than the number of elements in the list,
	// just treat this like an ordinary append, using the exported AddItem()
	if index >= len(l.items) {
		return l.AddItem(mainText, secondaryText, selected)
	}

	newItem := &mediaItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Selected:      selected,
	}

	l.items = append(l.items, nil)           // add a nil item to make room in the buffer for newItem
	copy(l.items[index+1:], l.items[index:]) // shift all items right, starting from insertion index
	l.items[index] = newItem                 // update the nil item at the insertion index

	if l.currentItem >= len(l.items) {
		l.currentItem = len(l.items) - 1
	}
	if len(l.items) == 1 && l.changed != nil {
		item := l.items[0]
		l.changed(0, item.MainText, item.SecondaryText)
	}
	return l
}

// GetItemCount returns the number of items in the list.
func (l *Browser) GetItemCount() int {
	return len(l.items)
}

// GetItemText returns an item's texts (main and secondary). Panics if the index
// is out of range.
func (l *Browser) GetItemText(index int) (main, secondary string) {
	return l.items[index].MainText, l.items[index].SecondaryText
}

// SetItemText sets an item's main and secondary text. Panics if the index is
// out of range.
func (l *Browser) SetItemText(index int, main, secondary string) *Browser {
	item := l.items[index]
	item.MainText = main
	item.SecondaryText = secondary
	return l
}

// Clear removes all items from the list.
func (l *Browser) Clear() *Browser {
	l.items = nil
	l.currentItem = 0
	return l
}

// Draw draws this primitive onto the screen.
func (l *Browser) Draw(screen tcell.Screen) {

	// check if a given value exists within a given closed interval by returning
	// a value less than zero, greater than zero, or equal to zero if the value
	// is less than the range minimum, greater than the range maximum, or if the
	// value exists in the interval (inclusive), respectively.
	contains := func(item, lo, hi int) int {
		switch {
		case item < lo:
			return -1
		case item > hi:
			return 1
		}
		return 0
	}

	l.Box.Draw(screen)

	// Determine the dimensions.
	x, y, width, height := l.GetInnerRect()
	yMax := y + height

	itemHeight := 1
	if l.showSecondaryText {
		itemHeight = 2
	}
	itemsPerPage := height / itemHeight

	// We want to keep the current selection in view. What is our offset?
	pos := contains(l.currentItem, l.viewOffset, l.viewOffset+itemsPerPage-1)
	switch {
	case pos < 0:
		l.viewOffset = l.currentItem
	case pos > 0:
		l.viewOffset = l.currentItem - (itemsPerPage - 1)
	default:
		// Adjust the viewing window if and only if our current position is not
		// inside the range of what's currently visible. Otherwise, let the user
		// navigate the list items freely, as in this default case.
	}

	// Draw the list items.
	for index, item := range l.items {
		if index < l.viewOffset {
			continue
		}

		if y >= yMax {
			break
		}

		// Main text.
		tview.Print(screen, item.MainText, x, y, width, tview.AlignLeft, l.mainTextColor)

		// Background color of selected text.
		if index == l.currentItem && (!l.selectedFocusOnly || l.HasFocus()) {
			for bx := 0; bx < width; bx++ {
				m, c, style, _ := screen.GetContent(x+bx, y)
				fg, _, _ := style.Decompose()
				if fg == l.mainTextColor {
					fg = l.selectedTextColor
				}
				style = style.Background(l.selectedBackgroundColor).Foreground(fg)
				screen.SetContent(x+bx, y, m, c, style)
			}
		}

		y++

		if y >= yMax {
			break
		}

		// Secondary text.
		if l.showSecondaryText {
			tview.Print(screen, item.SecondaryText, x, y, width, tview.AlignLeft, l.secondaryTextColor)
			y++
		}
	}
}

// InputHandler returns the handler for this primitive.
func (l *Browser) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return l.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		previousItem := l.currentItem

		switch key := event.Key(); key {
		case tcell.KeyTab, tcell.KeyDown, tcell.KeyRight:
			l.currentItem++
		case tcell.KeyBacktab, tcell.KeyUp, tcell.KeyLeft:
			l.currentItem--
		case tcell.KeyHome:
			l.currentItem = 0
		case tcell.KeyEnd:
			l.currentItem = len(l.items) - 1
		case tcell.KeyPgDn:
			l.currentItem += 5
		case tcell.KeyPgUp:
			l.currentItem -= 5
		case tcell.KeyEnter:
			if l.currentItem >= 0 && l.currentItem < len(l.items) {
				item := l.items[l.currentItem]
				if item.Selected != nil {
					item.Selected()
				}
				if l.selected != nil {
					l.selected(l.currentItem, item.MainText, item.SecondaryText)
				}
			}
		case tcell.KeyEscape:
			if l.done != nil {
				l.done()
			}
		}

		if l.currentItem < 0 {
			l.currentItem = len(l.items) - 1
		} else if l.currentItem >= len(l.items) {
			l.currentItem = 0
		}

		if l.currentItem != previousItem && l.currentItem < len(l.items) && l.changed != nil {
			item := l.items[l.currentItem]
			l.changed(l.currentItem, item.MainText, item.SecondaryText)
		}
	})
}
