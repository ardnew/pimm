// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 01 Jan 2019
//  FILE: browser.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    based on tview.List, this implements all methods of the tview.Primitive
//    interface. this affords an easier means of changing the appearance and
//    behaviors of the tview.List widget used as the primary interface for
//    presenting media lists to the user.
//
// =============================================================================

package main

import (
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// local unexported constants for the Browser primitive.
const (
	invalidIndex = -1
)

// mediaItem represents one Media object in a Browser.
type mediaItem struct {
	*Media                 // the corresponding Media item represented by this object.
	SourceLibrary *Library // the Library collection in which this item was found.
	Owner         *Browser // the Browser list in which this item is a member.
	MainText      string   // The main text of the list item.
	SecondaryText string   // A secondary text to be shown underneath the main text.
	Selected      func()   // The optional function which is called when the item is selected.
}

// function isValidIndex() checks if a given index is valid (in-range) for the
// provided mediaItem slice.
func isValidIndex(l []*mediaItem, i int) bool {
	return invalidIndex != i && i >= 0 && i < len(l)
}

// function findItem() attempts to locate the receiver in a given mediaItem
// slice and returns the first index in which it is found along with a boolean
// flag indicating if the item was found (true) or not (false).
func (m *mediaItem) findItem(l []*mediaItem) (int, bool) {
	for i, a := range l {
		if a == m {
			return i, true
		}
	}
	return invalidIndex, false
}

// function hideItem() removes a mediaItem from its owner Browser by removing it
// from the list of items displayed to the user and appending it to a separate
// private list of items which are not drawn on screen. items hidden in this way
// can be restored to visibility by calling it's companion function showItem().
func (l *mediaItem) hideItem() *mediaItem {

	// verify our item doesn't already exist in the list of hidden items.
	_, isHidden := l.findItem(l.Owner.hiddenItem)

	// append to the hidden items list only if it doesn't exist there already.
	if !isHidden {
		l.Owner.hiddenItem = append(l.Owner.hiddenItem, l)

		// theoretically, the only way an item could exist in the visible items
		// list is if it doesn't exist in the hidden items list. so only in this
		// case do we try to find its index among the visible items.
		visibleIndex, isVisible := l.findItem(l.Owner.visibleItem)

		// and finally, if found, remove it from the list of visible items.
		if isVisible {
			l.Owner.removeItem(visibleIndex)
		}
	}

	// if the item exists in the hidden items list, we currently do not even
	// attempt to locate it in the visible items list. this could be a mistake,
	// but at the moment it saves us some processing cycles (for a full linear
	// search) as it logically makes no sense for this condition to exist.

	return l
}

// function showItem() restores an item that was hidden with the companion
// function hideItem() by removing it from the private hidden items list and
// inserting it back into the list of visible items in the correct position.
func (l *mediaItem) showItem() *mediaItem {

	// find our current item in the list of hidden items.
	hiddenIndex, isHidden := l.findItem(l.Owner.hiddenItem)

	// verify the item was hidden before trying to reinsert it back into the
	// visible items list. there's no reason it shouldn't be in the hidden items
	// list unless its already in the visible items list.
	if isHidden {
		// remove the item by appending the items trailing our hidden item index
		// to the items preceding it, replacing our hidden items list with the
		// result of this list concatenation.
		l.Owner.hiddenItem =
			append(l.Owner.hiddenItem[:hiddenIndex], l.Owner.hiddenItem[hiddenIndex+1:]...)

			// and finally, locate the correct insertion position in the visible
			// items list for the previously hidden item, as other items may have
			// been added or removed since this item was last hidden.
		position, primary, secondary := l.Owner.positionForMediaItem(l.Media)
		l.Owner.insertMediaItem(l.SourceLibrary, l.Media, position, primary, secondary, l.Selected)
	}

	// if the item does not exist in the hidden items list, then do not attempt
	// to insert it into the visible items list. this may be a mistake, but at
	// the moment it saves us some processing cycles (for a full linear search)
	// as it logically makes no sense for this condition to exist.

	return l
}

// Browser displays rows of items, each of which can be selected.
type Browser struct {
	*tview.Box

	// The visible items of the list.
	visibleItem []*mediaItem

	// The hidden items of the list.
	hiddenItem []*mediaItem

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

// newBrowser returns a new form.
func newBrowser() *Browser {
	return &Browser{
		Box:                     tview.NewBox(),
		visibleItem:             []*mediaItem{},
		hiddenItem:              []*mediaItem{},
		showSecondaryText:       true,
		mainTextColor:           colorScheme.activeText,
		secondaryTextColor:      colorScheme.inactiveText,
		selectedTextColor:       colorScheme.backgroundPrimary,
		selectedBackgroundColor: colorScheme.highlightPrimary,
	}
}

// function showLibrary() filters the list of data items shown in the Browser on
// a per-library basis. it accepts no other parameters and uses no other
// criteria to determine which data items to display. if a Library is provided,
// then only the items which are members of that library will be displayed. if
// a nil value is provided (the default), then all data items from all libraries
// are displayed.
func (l *Browser) showLibrary(library *Library) {

	// create a single slice containing -all- items for simpler traversal of all
	// candidates.
	allItems := []*mediaItem{}
	allItems = append(allItems, l.hiddenItem...)
	allItems = append(allItems, l.visibleItem...)

	// check if we are intending to filter the items
	if nil == library {
		// a nil library means no filtering, display all data items from all
		// libraries.
		for _, m := range allItems {
			m.showItem()
		}
	} else {
		//
		// walk backward over the list so that we do not skip an item when the
		// list items are shifted due to a removal. for example, with items to
		// be removed enclosed in curly braces, a forward traversal would result
		// in the following iterations and resulting list:
		//
		//  iter#:  list[]                       description
		//  ------  ---------------------------  -------------------------------
		//          [  A , {B}, {C},  D ,  E  ]
		//      0:     ^                         first item "A" left as-is, not a candidate for removal
		//          [  A , {B}, {C},  D ,  E  ]
		//      1:          ^                    next item "B" removed, shift trailing items to the left
		//          [  A , {C},  D ,  E       ]
		//      2:               ^               next item "D" left as-is, not a candidate for removal
		//          [  A , {C},  D ,  E       ]
		//      3:                    ^          last item "E" left as-is, not a candidate for removal, terminate loop
		//
		// so in this example, item "C" originally at list index 2 is skipped
		// during a forward traversal and remains at list index 1 once the
		// traversal finishes. traversing the list from the opposite direction
		// prevents not-yet-examined items from being moved to positions that
		// have already been evaluated.
		//
		for i := len(allItems) - 1; i >= 0; i-- {
			m := allItems[i]
			if m.SourceLibrary != library {
				m.hideItem()
			} else {
				m.showItem()
			}
		}
	}
}

// setCurrentItem sets the currently selected item by its index. This triggers
// a "changed" event.
func (l *Browser) setCurrentItem(index int) *Browser {
	l.currentItem = index
	if l.currentItem < len(l.visibleItem) && l.changed != nil {
		item := l.visibleItem[l.currentItem]
		l.changed(l.currentItem, item.MainText, item.SecondaryText)
	}
	return l
}

// getCurrentItem returns the index of the currently selected list item.
func (l *Browser) getCurrentItem() int {
	return l.currentItem
}

// setMainTextColor sets the color of the items' main text.
func (l *Browser) setMainTextColor(color tcell.Color) *Browser {
	l.mainTextColor = color
	return l
}

// setSecondaryTextColor sets the color of the items' secondary text.
func (l *Browser) setSecondaryTextColor(color tcell.Color) *Browser {
	l.secondaryTextColor = color
	return l
}

// setSelectedTextColor sets the text color of selected items.
func (l *Browser) setSelectedTextColor(color tcell.Color) *Browser {
	l.selectedTextColor = color
	return l
}

// setSelectedBackgroundColor sets the background color of selected items.
func (l *Browser) setSelectedBackgroundColor(color tcell.Color) *Browser {
	l.selectedBackgroundColor = color
	return l
}

// setSelectedFocusOnly sets a flag which determines when the currently selected
// list item is highlighted. If set to true, selected items are only highlighted
// when the list has focus. If set to false, they are always highlighted.
func (l *Browser) setSelectedFocusOnly(focusOnly bool) *Browser {
	l.selectedFocusOnly = focusOnly
	return l
}

// setShowSecondaryText determines whether or not to show secondary item texts.
func (l *Browser) setShowSecondaryText(show bool) *Browser {
	l.showSecondaryText = show
	return l
}

// setChangedFunc sets the function which is called when the user navigates to
// a list item. The function receives the item's index in the list of items
// (starting with 0), its main text, and its secondary text.
//
// This function is also called when the first item is added or when
// setCurrentItem() is called.
func (l *Browser) setChangedFunc(handler func(index int, mainText string, secondaryText string)) *Browser {
	l.changed = handler
	return l
}

// setSelectedFunc sets the function which is called when the user selects a
// list item by pressing Enter on the current selection. The function receives
// the item's index in the list of items (starting with 0), its main text, and
// its secondary text.
func (l *Browser) setSelectedFunc(handler func(int, string, string)) *Browser {
	l.selected = handler
	return l
}

// setDoneFunc sets a function which is called when the user presses the Escape
// key.
func (l *Browser) setDoneFunc(handler func()) *Browser {
	l.done = handler
	return l
}

// removeItem removes the item with the given index (starting at 0) from the
// list. Does nothing if the index is out of range. This triggers a "changed"
// event if and only if the currently selected item is changed because of the
// removal.
func (l *Browser) removeItem(index int) *Browser {
	if index < 0 || index >= len(l.visibleItem) {
		return l
	}
	l.visibleItem = append(l.visibleItem[:index], l.visibleItem[index+1:]...)

	// calculate the new length after removal of the item (this should probably
	// always be the previous length - 1, obviously, I think...)
	length := len(l.visibleItem)

	// determine if we should trigger the "changed" event by determining if our
	// currently selected item is outside the range of the list after item
	// removal (and the resulting list is non-empty) or if the item removed is
	// the currently selected item. in either case, the currently selected item
	// must be changed. in particular for these cases, it must be decremented.
	changeCurrentItem :=
		(l.currentItem >= length && length > 0) || (l.currentItem == index)

	// if we've determined the currently selected item should be changed, then
	// perform the change (clamped to list domain [0, length-1]) and trigger
	// a "changed" event.
	if changeCurrentItem {
		l.currentItem--
		if l.currentItem < 0 {
			l.currentItem = 0
		} else if l.currentItem >= length {
			l.currentItem = length - 1
		}
		if nil != l.changed {
			item := l.visibleItem[l.currentItem]
			l.changed(l.currentItem, item.MainText, item.SecondaryText)
		}
	}

	return l
}

// function positionForMediaItem() iterates over the visible items in the media
// item browser to decide which position the provided media item name and path
// should be inserted and formats the text to be displayed in both primary and
// secondary text strings. this method effectively provides the sorting order of
// the media item library.
func (l *Browser) positionForMediaItem(media *Media) (int, string, string) {

	// determines WHEN the discovered item (discoName, discoPath) should be
	// inserted based on the current item (currName, currPath) iteration.
	shouldInsert := func(discoName, discoPath, currName, currPath string) bool {

		// sorted by name
		return (currName == discoName && currPath >= discoPath) || (currName >= discoName)

		// sorted by path
		//return (currPath == discoPath && currName >= discoName) || (currPath >= discoPath)
	}

	// the formatting/appearance to use for the item's displayed text.
	fmtPrimary := func(m *Media) string { return m.AbsName }
	fmtSecondary := func(m *Media) string { return m.AbsPath }

	primary := fmtPrimary(media)
	secondary := fmtSecondary(media)

	// append by default, because we did not find an item that already exists in
	// our list which should appear after our new item we are trying to insert
	// -- i.e. the new item is lexicographically last.
	var position int = l.getItemCount()
	if numItems := position; numItems > 0 {
		for i := 0; i < numItems; i++ {

			itemName, itemPath := l.getItemText(i)

			insert := shouldInsert(
				strings.ToUpper(primary),
				strings.ToUpper(secondary),
				strings.ToUpper(itemName),
				strings.ToUpper(itemPath))

			if insert {
				position = i
				break
			}
		}
	}
	return position, primary, secondary
}

// addMediaItem adds a new item to the list. An item has a main text which will
// be highlighted when selected. It also has a secondary text which is shown
// underneath the main text (if it is set to visible) but which may remain
// empty.
//
// The "selected" callback will be invoked when the user selects the item. You
// may provide nil if no such item is needed or if all events are handled
// through the selected callback set with setSelectedFunc().
func (l *Browser) addMediaItem(library *Library, media *Media, mainText, secondaryText string, selected func()) *Browser {

	l.visibleItem = append(l.visibleItem, &mediaItem{
		Media:         media,
		SourceLibrary: library,
		Owner:         l,
		MainText:      mainText,
		SecondaryText: secondaryText,
		Selected:      selected,
	})
	if len(l.visibleItem) == 1 && l.changed != nil {
		item := l.visibleItem[0]
		l.changed(0, item.MainText, item.SecondaryText)
	}
	return l
}

// function insertMediaItem() accepts all parameters necessary to define a media
// item, creates it, and then inserts it into the list at the correct position
// among -all- data items. this is effectively an insertion sort implemented
// with linear traversal -- so not the fastest, but simple and effective.
func (l *Browser) insertMediaItem(library *Library, media *Media, index int, mainText, secondaryText string, selected func()) *Browser {

	// several different ways to interpret index < 0. one convenient way would
	// be to insert starting from the end of the list. the safest option, which
	// is implemented here, is to just consider it as invalid input and return
	// the original list unmodified.
	if index < 0 {
		return l
	}

	// if the index provided is greater than the number of elements in the list,
	// then treat this like an ordinary append using the exported addMediaItem()
	if index >= len(l.visibleItem) {
		return l.addMediaItem(library, media, mainText, secondaryText, selected)
	}

	newItem := &mediaItem{
		Media:         media,
		SourceLibrary: library,
		Owner:         l,
		MainText:      mainText,
		SecondaryText: secondaryText,
		Selected:      selected,
	}

	l.visibleItem = append(l.visibleItem, nil)           // add a nil item to make room in the buffer for newItem
	copy(l.visibleItem[index+1:], l.visibleItem[index:]) // shift all items right, starting from insertion index
	l.visibleItem[index] = newItem                       // update the nil item at the insertion index

	// if our currently selected item is beyond the end of our slice, change it
	// to be the last element of the slice.
	if l.currentItem >= len(l.visibleItem) {
		l.currentItem = len(l.visibleItem) - 1
	}

	// if this is the first item we've added to the list, trigger a "changed"
	// event.
	if len(l.visibleItem) == 1 && l.changed != nil {
		item := l.visibleItem[0]
		l.changed(0, item.MainText, item.SecondaryText)
	}
	return l
}

// getItemCount returns the number of items in the list.
func (l *Browser) getItemCount() int {
	return len(l.visibleItem)
}

// getItemText returns an item's texts (main and secondary). Panics if the index
// is out of range.
func (l *Browser) getItemText(index int) (main, secondary string) {
	return l.visibleItem[index].MainText, l.visibleItem[index].SecondaryText
}

// setItemText sets an item's main and secondary text. Panics if the index is
// out of range.
func (l *Browser) setItemText(index int, main, secondary string) *Browser {
	item := l.visibleItem[index]
	item.MainText = main
	item.SecondaryText = secondary
	return l
}

// clear removes all items from the list.
func (l *Browser) clear() *Browser {
	l.visibleItem = nil
	l.hiddenItem = nil
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

	// ensure the embedded Box primitive drawing occurs for all of the basic
	// appearance logic.
	l.Box.Draw(screen)

	// Determine the dimensions.
	x, y, width, height := l.GetInnerRect()
	yMax := y + height

	// the height of our data items list elements affects how many data items we
	// can fit on screen at any one time.
	itemHeight := 1
	if l.showSecondaryText {
		itemHeight = 2
	}
	itemsPerPage := height / itemHeight

	// we want to keep the current selection in view. What is our offset? check
	// if our current selection lies within the range of our current view offset
	// and the offset plus number of items we can fit on screen. if so, then do
	// not adjust the displayed items. if not, then determine which direction
	// we need to scroll so that it does.
	pos := contains(l.currentItem, l.viewOffset, l.viewOffset+itemsPerPage-1)
	switch {
	case pos < 0:
		// our current selection exists prior all items currently visible. so
		// scroll the view up so that our currently selected item exists as the
		// top-most visible item.
		l.viewOffset = l.currentItem
	case pos > 0:
		// our current selection exists following all items currently visible.
		// so scroll the view down so that our currently selected item exists as
		// the bottom-most visible item.
		l.viewOffset = l.currentItem - (itemsPerPage - 1)
	default:
		// Adjust the viewing window if and only if our current position is not
		// inside the range of what's currently visible. Otherwise, let the user
		// navigate the list items freely, as in this default case -- we are not
		// scrolling in any direction, only the current selection highlight is
		// being moved on screen.
	}

	// iterate over all possible items in the list of visible items, drawing
	// only those that lie within the range of what's viewable on screen.
	for index, item := range l.visibleItem {

		// skip all data items prior our view offset.
		if index < l.viewOffset {
			continue
		}

		// bail out once we've traversed beyond our current offset plus number
		// of items we can fit on screen. all visible items have already been
		// drawn by this point in the list traversal.
		if y >= yMax {
			break
		}

		// Main text.
		tview.Print(screen, item.MainText, x, y, width, tview.AlignLeft, l.mainTextColor)

		// Background color of selected text.
		if index == l.currentItem && (!l.selectedFocusOnly || l.HasFocus()) {
			// we have to color each individual cell of the current row, so we
			// iterate over each column.
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

		// increment our current list traversal offset.
		y++

		// Secondary text.
		if l.showSecondaryText {

			// a second check to bail out of the list traversal in the event
			// that only part of the data item can be drawn on screen.
			if y >= yMax {
				break
			}
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
			l.currentItem = len(l.visibleItem) - 1
		case tcell.KeyPgDn:
			l.currentItem += 5
		case tcell.KeyPgUp:
			l.currentItem -= 5
		case tcell.KeyEnter:
			if l.currentItem >= 0 && l.currentItem < len(l.visibleItem) {
				item := l.visibleItem[l.currentItem]
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
			l.currentItem = len(l.visibleItem) - 1
		} else if l.currentItem >= len(l.visibleItem) {
			l.currentItem = 0
		}

		if l.currentItem != previousItem && l.currentItem < len(l.visibleItem) && l.changed != nil {
			item := l.visibleItem[l.currentItem]
			l.changed(l.currentItem, item.MainText, item.SecondaryText)
		}
	})
}
