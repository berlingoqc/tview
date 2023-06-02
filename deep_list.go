package tview

import (
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type subList struct {
	display bool
	items   []*deepListItem
}

// deepListItem represents one item in a DeepList.
type deepListItem struct {
	MainText      string // The main text of the list item.
	SecondaryText string // A secondary text to be shown underneath the main text.
	Shortcut      rune   // The key to select the list item directly, 0 if there is no shortcut.
	Selected      func() // The optional function which is called when the item is selected.

	SubList *subList // The sublist
}

// DeepList displays rows of items, each of which can be selected. DeepList items can be
// shown as a single line or as two lines. They can be selected by pressing
// their assigned shortcut key, navigating to them and pressing Enter, or
// clicking on them with the mouse. The following key binds are available:
//
//   - Down arrow / tab: Move down one item.
//   - Up arrow / backtab: Move up one item.
//   - Home: Move to the first item.
//   - End: Move to the last item.
//   - Page down: Move down one page.
//   - Page up: Move up one page.
//   - Enter / Space: Select the current item.
//   - Right / left: Scroll horizontally. Only if the list is wider than the
//     available space.
//
// See [DeepList.SetChangedFunc] for a way to be notified when the user navigates
// to a list item. See [DeepList.SetSelectedFunc] for a way to be notified when a
// list item was selected.
//
// See https://github.com/rivo/tview/wiki/DeepList for an example.
type DeepList struct {
	*Box

	// The items of the list.
	items []*deepListItem

	// The index of the currently selected item.
	currentItem []int

	// Whether or not to show the secondary item texts.
	showSecondaryText bool

	// The item main text style.
	mainTextStyle tcell.Style

	// The item secondary text style.
	secondaryTextStyle tcell.Style

	// The item shortcut text style.
	shortcutStyle tcell.Style

	// The style for selected items.
	selectedStyle tcell.Style

	// If true, the selection is only shown when the list has focus.
	selectedFocusOnly bool

	// If true, the entire row is highlighted when selected.
	highlightFullLine bool

	// Whether or not navigating the list will wrap around.
	wrapAround bool

	// The number of list items skipped at the top before the first item is
	// drawn.
	itemOffset int

	// The number of cells skipped on the left side of an item text. Shortcuts
	// are not affected.
	horizontalOffset int

	// Set to true if a currently visible item flows over the right border of
	// the box. This is set by the Draw() function. It determines the behaviour
	// of the right arrow key.
	overflowing bool

	// An optional function which is called when the user has navigated to a
	// list item.
	changed func(indexes []int, mainText, secondaryText string, shortcut rune)

	// An optional function which is called when a list item was selected. This
	// function will be called even if the list item defines its own callback.
	selected func(index []int, mainText, secondaryText string, shortcut rune)

	// An optional function which is called when the user presses the Escape key.
	done func()
}

// NewDeepList returns a new list.
func NewDeepList() *DeepList {
	return &DeepList{
		Box:                NewBox(),
		showSecondaryText:  true,
		wrapAround:         true,
		currentItem:        []int{0},
		mainTextStyle:      tcell.StyleDefault.Foreground(Styles.PrimaryTextColor),
		secondaryTextStyle: tcell.StyleDefault.Foreground(Styles.TertiaryTextColor),
		shortcutStyle:      tcell.StyleDefault.Foreground(Styles.SecondaryTextColor),
		selectedStyle:      tcell.StyleDefault.Foreground(Styles.PrimitiveBackgroundColor).Background(Styles.PrimaryTextColor),
	}
}

func parseIndexes(index int, indexes []int, items []*deepListItem) []int {
	if indexes[index] < 0 {
		indexes[index] = len(items) + indexes[index]
	}
	if indexes[index] >= len(items) {
		index = len(items) - 1
	}
	if index < 0 {
		index = 0
	}

	if index+1 >= len(indexes) {
		return indexes
	}

	item := items[index]

	if item.SubList == nil || len(item.SubList.items) == 0 {
		return indexes
	}

	return parseIndexes(index+1, indexes, item.SubList.items)
}

func getItem(index int, indexes []int, items []*deepListItem) (*deepListItem, []*deepListItem) {
	item := items[index]

	if index+1 >= len(indexes) {
		return item, items
	}

	if item.SubList == nil || len(item.SubList.items) == 0 {
		return item, items
	}

	if item.SubList.display == false {
		item.SubList.display = true
	}

	return getItem(index+1, indexes, item.SubList.items)
}

func removeItem(index int, indexes []int, items []*deepListItem) int {
	item := items[index]

	if index+1 >= len(indexes) {
		items = append(items[:indexes[index]], items[indexes[index]+1:]...)

		return len(items)
	}

	if item.SubList == nil || len(item.SubList.items) == 0 {
		return len(items)
	}

	return removeItem(index+1, indexes, item.SubList.items)
}

func moveSelectionIndex(index int, indexes []int, items []*deepListItem, change int) []int {

	item, _ := getItem(0, indexes, items)

	log.Printf("item is %s %v \n", item.MainText, item.SubList != nil && item.SubList.display)

	if change > 0 {
		// go deep or go to next
		if item.SubList != nil && item.SubList.display && len(item.SubList.items) > 0 {
			indexes = append(indexes, 0)

			log.Printf("deep : %d \n", indexes)
		} else {
			// go to next from parent array
			indexes[len(indexes)-1] += change
			log.Printf("increment : %d \n", indexes)
		}

		return indexes
	} else {
		indexes[len(indexes)-1] += change
		log.Printf("decrement : %d \n", indexes)
		return indexes
	}
}

// TODO: this code wont work properly in complexe case
func getOffset(total int, index int, indexes []int, items []*deepListItem) int {
	log.Printf("index : %d %v %d \n", index, indexes, len(items))
	for _, v := range items[:indexes[index]] {
		total += 1
		if v.SubList != nil && v.SubList.display && len(v.SubList.items) > 0 {
			total = getOffset(total, index+1, indexes, v.SubList.items)
		}
	}
	return total
}

// TODO: move me
func equals(ai []int, b []int) bool {
	for i, v := range b {
		if ai[i] != v {
			return false
		}
	}
	return true
}

// SetCurrentItem sets the currently selected item by its index, starting at 0
// for the first item. If a negative index is provided, items are referred to
// from the back (-1 = last item, -2 = second-to-last item, and so on). Out of
// range indices are clamped to the beginning/end.
//
// Calling this function triggers a "changed" event if the selection changes.
func (l *DeepList) SetCurrentItem(indexes []int) *DeepList {

	indexes = parseIndexes(0, indexes, l.items)

	if !equals(indexes, l.currentItem) && l.changed != nil {
		item, _ := getItem(0, indexes, l.items)
		l.changed(indexes, item.MainText, item.SecondaryText, item.Shortcut)
	}

	l.currentItem = indexes

	l.adjustOffset()

	return l
}

// GetCurrentItem returns the index of the currently selected list item,
// starting at 0 for the first item.
func (l *DeepList) GetCurrentItem() []int {
	return l.currentItem
}

// SetOffset sets the number of items to be skipped (vertically) as well as the
// number of cells skipped horizontally when the list is drawn. Note that one
// item corresponds to two rows when there are secondary texts. Shortcuts are
// always drawn.
//
// These values may change when the list is drawn to ensure the currently
// selected item is visible and item texts move out of view. Users can also
// modify these values by interacting with the list.
func (l *DeepList) SetOffset(items, horizontal int) *DeepList {
	l.itemOffset = items
	l.horizontalOffset = horizontal
	return l
}

// GetOffset returns the number of items skipped while drawing, as well as the
// number of cells item text is moved to the left. See also SetOffset() for more
// information on these values.
func (l *DeepList) GetOffset() (int, int) {
	return l.itemOffset, l.horizontalOffset
}

// RemoveItem removes the item with the given index (starting at 0) from the
// list. If a negative index is provided, items are referred to from the back
// (-1 = last item, -2 = second-to-last item, and so on). Out of range indices
// are clamped to the beginning/end, i.e. unless the list is empty, an item is
// always removed.
//
// The currently selected item is shifted accordingly. If it is the one that is
// removed, a "changed" event is fired, unless no items are left.
func (l *DeepList) RemoveItem(indexes []int) *DeepList {
	if len(l.items) == 0 {
		return l
	}

	// Adjust index.
	indexes = parseIndexes(0, indexes, l.items)

	// Remove item.
	lenAfter := removeItem(0, indexes, l.items)

	// If there is nothing left, we're done.
	if lenAfter == 0 {
		return l
	}

	// Shift current item.
	//previousCurrentItem := l.currentItem
	//if l.currentItem > index || l.currentItem == len(l.items) {
	//	l.currentItem--
	//}

	// Fire "changed" event for removed items.
	//if previousCurrentItem == index && l.changed != nil {
	//	item := l.items[l.currentItem]
	//	l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
	//}

	return l
}

// SetMainTextColor sets the color of the items' main text.
func (l *DeepList) SetMainTextColor(color tcell.Color) *DeepList {
	l.mainTextStyle = l.mainTextStyle.Foreground(color)
	return l
}

// SetMainTextStyle sets the style of the items' main text. Note that the
// background color is ignored in order not to override the background color of
// the list itself.
func (l *DeepList) SetMainTextStyle(style tcell.Style) *DeepList {
	l.mainTextStyle = style
	return l
}

// SetSecondaryTextColor sets the color of the items' secondary text.
func (l *DeepList) SetSecondaryTextColor(color tcell.Color) *DeepList {
	l.secondaryTextStyle = l.secondaryTextStyle.Foreground(color)
	return l
}

// SetSecondaryTextStyle sets the style of the items' secondary text. Note that
// the background color is ignored in order not to override the background color
// of the list itself.
func (l *DeepList) SetSecondaryTextStyle(style tcell.Style) *DeepList {
	l.secondaryTextStyle = style
	return l
}

// SetShortcutColor sets the color of the items' shortcut.
func (l *DeepList) SetShortcutColor(color tcell.Color) *DeepList {
	l.shortcutStyle = l.shortcutStyle.Foreground(color)
	return l
}

// SetShortcutStyle sets the style of the items' shortcut. Note that the
// background color is ignored in order not to override the background color of
// the list itself.
func (l *DeepList) SetShortcutStyle(style tcell.Style) *DeepList {
	l.shortcutStyle = style
	return l
}

// SetSelectedTextColor sets the text color of selected items. Note that the
// color of main text characters that are different from the main text color
// (e.g. color tags) is maintained.
func (l *DeepList) SetSelectedTextColor(color tcell.Color) *DeepList {
	l.selectedStyle = l.selectedStyle.Foreground(color)
	return l
}

// SetSelectedBackgroundColor sets the background color of selected items.
func (l *DeepList) SetSelectedBackgroundColor(color tcell.Color) *DeepList {
	l.selectedStyle = l.selectedStyle.Background(color)
	return l
}

// SetSelectedStyle sets the style of the selected items. Note that the color of
// main text characters that are different from the main text color (e.g. color
// tags) is maintained.
func (l *DeepList) SetSelectedStyle(style tcell.Style) *DeepList {
	l.selectedStyle = style
	return l
}

// SetSelectedFocusOnly sets a flag which determines when the currently selected
// list item is highlighted. If set to true, selected items are only highlighted
// when the list has focus. If set to false, they are always highlighted.
func (l *DeepList) SetSelectedFocusOnly(focusOnly bool) *DeepList {
	l.selectedFocusOnly = focusOnly
	return l
}

// SetHighlightFullLine sets a flag which determines whether the colored
// background of selected items spans the entire width of the view. If set to
// true, the highlight spans the entire view. If set to false, only the text of
// the selected item from beginning to end is highlighted.
func (l *DeepList) SetHighlightFullLine(highlight bool) *DeepList {
	l.highlightFullLine = highlight
	return l
}

// ShowSecondaryText determines whether or not to show secondary item texts.
func (l *DeepList) ShowSecondaryText(show bool) *DeepList {
	l.showSecondaryText = show
	return l
}

// SetWrapAround sets the flag that determines whether navigating the list will
// wrap around. That is, navigating downwards on the last item will move the
// selection to the first item (similarly in the other direction). If set to
// false, the selection won't change when navigating downwards on the last item
// or navigating upwards on the first item.
func (l *DeepList) SetWrapAround(wrapAround bool) *DeepList {
	l.wrapAround = wrapAround
	return l
}

// SetChangedFunc sets the function which is called when the user navigates to
// a list item. The function receives the item's index in the list of items
// (starting with 0), its main text, secondary text, and its shortcut rune.
//
// This function is also called when the first item is added or when
// SetCurrentItem() is called.
func (l *DeepList) SetChangedFunc(handler func(indexes []int, mainText string, secondaryText string, shortcut rune)) *DeepList {
	l.changed = handler
	return l
}

// SetSelectedFunc sets the function which is called when the user selects a
// list item by pressing Enter on the current selection. The function receives
// the item's index in the list of items (starting with 0), its main text,
// secondary text, and its shortcut rune.
func (l *DeepList) SetSelectedFunc(handler func([]int, string, string, rune)) *DeepList {
	l.selected = handler
	return l
}

// SetDoneFunc sets a function which is called when the user presses the Escape
// key.
func (l *DeepList) SetDoneFunc(handler func()) *DeepList {
	l.done = handler
	return l
}

// AddItem calls InsertItem() with an index of -1.
func (l *DeepList) AddItem(mainText, secondaryText string, shortcut rune, selected func()) *DeepList {
	l.InsertItem(-1, mainText, secondaryText, shortcut, selected)
	return l
}

func (l *DeepList) AddSubItem(mainText, secondaryText string, shortcut rune, display bool, selected func()) *DeepList {
	lastIndex := len(l.items) - 1
	if lastIndex < 0 {
		return l
	}

	item := &deepListItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		Selected:      selected,
	}

	parentItem := l.items[lastIndex]
	if parentItem.SubList == nil {
		parentItem.SubList = &subList{

			display: display,
			items:   make([]*deepListItem, 1),
		}
		parentItem.SubList.items[0] = item
	} else {
		parentItem.SubList.display = display
		parentItem.SubList.items = append(parentItem.SubList.items, item)
	}

	return l
}

func (l *DeepList) ToggleSubListDisplay(index int) *DeepList {
	if index >= len(l.items) {
		return l
	}
	if l.items[index].SubList == nil {
		return l
	}

	l.items[index].SubList.display = !l.items[index].SubList.display

	return l
}

// InsertItem adds a new item to the list at the specified index. An index of 0
// will insert the item at the beginning, an index of 1 before the second item,
// and so on. An index of GetItemCount() or higher will insert the item at the
// end of the list. Negative indices are also allowed: An index of -1 will
// insert the item at the end of the list, an index of -2 before the last item,
// and so on. An index of -GetItemCount()-1 or lower will insert the item at the
// beginning.
//
// An item has a main text which will be highlighted when selected. It also has
// a secondary text which is shown underneath the main text (if it is set to
// visible) but which may remain empty.
//
// The shortcut is a key binding. If the specified rune is entered, the item
// is selected immediately. Set to 0 for no binding.
//
// The "selected" callback will be invoked when the user selects the item. You
// may provide nil if no such callback is needed or if all events are handled
// through the selected callback set with SetSelectedFunc().
//
// The currently selected item will shift its position accordingly. If the list
// was previously empty, a "changed" event is fired because the new item becomes
// selected.
func (l *DeepList) InsertItem(index int, mainText, secondaryText string, shortcut rune, selected func()) *DeepList {
	item := &deepListItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		Selected:      selected,
	}

	// Shift index to range.
	if index < 0 {
		index = len(l.items) + index + 1
	}
	if index < 0 {
		index = 0
	} else if index > len(l.items) {
		index = len(l.items)
	}

	// Shift current item.
	if l.currentItem[0] < len(l.items) && l.currentItem[0] >= index {
		l.currentItem[0]++
	}

	// Insert item (make space for the new item, then shift and insert).
	l.items = append(l.items, nil)
	if index < len(l.items)-1 { // -1 because l.items has already grown by one item.
		copy(l.items[index+1:], l.items[index:])
	}
	l.items[index] = item

	// Fire a "change" event for the first item in the list.
	if len(l.items) == 1 && l.changed != nil {
		item := l.items[0]
		l.changed([]int{0}, item.MainText, item.SecondaryText, item.Shortcut)
	}
	return l
}

// GetItemCount returns the number of items in the list.
func (l *DeepList) GetItemCount() int {
	// TODO: should return size of all display sub items
	return len(l.items)
}

// GetItemText returns an item's texts (main and secondary). Panics if the index
// is out of range.
func (l *DeepList) GetItemText(index int) (main, secondary string) {
	return l.items[index].MainText, l.items[index].SecondaryText
}

// SetItemText sets an item's main and secondary text. Panics if the index is
// out of range.
func (l *DeepList) SetItemText(index int, main, secondary string) *DeepList {
	item := l.items[index]
	item.MainText = main
	item.SecondaryText = secondary
	return l
}

// FindItems searches the main and secondary texts for the given strings and
// returns a list of item indices in which those strings are found. One of the
// two search strings may be empty, it will then be ignored. Indices are always
// returned in ascending order.
//
// If mustContainBoth is set to true, mainSearch must be contained in the main
// text AND secondarySearch must be contained in the secondary text. If it is
// false, only one of the two search strings must be contained.
//
// Set ignoreCase to true for case-insensitive search.
func (l *DeepList) FindItems(mainSearch, secondarySearch string, mustContainBoth, ignoreCase bool) (indices []int) {
	if mainSearch == "" && secondarySearch == "" {
		return
	}

	if ignoreCase {
		mainSearch = strings.ToLower(mainSearch)
		secondarySearch = strings.ToLower(secondarySearch)
	}

	for index, item := range l.items {
		mainText := item.MainText
		secondaryText := item.SecondaryText
		if ignoreCase {
			mainText = strings.ToLower(mainText)
			secondaryText = strings.ToLower(secondaryText)
		}

		// strings.Contains() always returns true for a "" search.
		mainContained := strings.Contains(mainText, mainSearch)
		secondaryContained := strings.Contains(secondaryText, secondarySearch)
		if mustContainBoth && mainContained && secondaryContained ||
			!mustContainBoth && (mainText != "" && mainContained || secondaryText != "" && secondaryContained) {
			indices = append(indices, index)
		}
	}

	return
}

// Clear removes all items from the list.
func (l *DeepList) Clear() *DeepList {
	l.items = nil
	l.currentItem = []int{0}
	return l
}

// Draw draws this primitive onto the screen.
func (l *DeepList) Draw(screen tcell.Screen) {
	l.Box.DrawForSubclass(screen, l)

	// Determine the dimensions.
	x, y, width, height := l.GetInnerRect()
	bottomLimit := y + height
	_, totalHeight := screen.Size()
	if bottomLimit > totalHeight {
		bottomLimit = totalHeight
	}

	// Do we show any shortcuts?
	var showShortcuts bool
	for _, item := range l.items {
		if item.Shortcut != 0 {
			showShortcuts = true
			x += 4
			width -= 4
			break
		}
	}

	if l.horizontalOffset < 0 {
		l.horizontalOffset = 0
	}

	// Draw the list items.
	var (
		maxWidth    int  // The maximum printed item width.
		overflowing bool // Whether a text's end exceeds the right border.
	)
PARENT:
	for index, item := range l.items {
		if index < l.itemOffset {
			continue
		}

		if y >= bottomLimit {
			break
		}

		// Shortcuts.
		if showShortcuts && item.Shortcut != 0 {
			printWithStyle(screen, fmt.Sprintf("(%s)", string(item.Shortcut)), x-5, y, 0, 4, AlignRight, l.shortcutStyle, true)
		}

		// Main text.
		_, printedWidth, _, end := printWithStyle(screen, item.MainText, x, y, l.horizontalOffset, width, AlignLeft, l.mainTextStyle, true)
		if printedWidth > maxWidth {
			maxWidth = printedWidth
		}
		if end < len(item.MainText) {
			overflowing = true
		}

		// Background color of selected text.
		if index == l.currentItem[0] && (!l.selectedFocusOnly || l.HasFocus()) {
			textWidth := width
			if !l.highlightFullLine {
				if w := TaggedStringWidth(item.MainText); w < textWidth {
					textWidth = w
				}
			}

			mainTextColor, _, _ := l.mainTextStyle.Decompose()
			for bx := 0; bx < textWidth; bx++ {
				m, c, style, _ := screen.GetContent(x+bx, y)
				fg, _, _ := style.Decompose()
				style = l.selectedStyle
				if fg != mainTextColor {
					style = style.Foreground(fg)
				}
				screen.SetContent(x+bx, y, m, c, style)
			}
		}
		y++

		if y >= bottomLimit {
			break
		}

		// Secondary text.
		if l.showSecondaryText {
			_, printedWidth, _, end := printWithStyle(screen, item.SecondaryText, x, y, l.horizontalOffset, width, AlignLeft, l.secondaryTextStyle, true)
			if printedWidth > maxWidth {
				maxWidth = printedWidth
			}
			if end < len(item.SecondaryText) {
				overflowing = true
			}

			y++
		}

		if item.SubList != nil && item.SubList.display {
			for _, subItem := range item.SubList.items {
				_, printedWidth, _, end := printWithStyle(screen, subItem.MainText, x, y, l.horizontalOffset, width, AlignLeft, l.secondaryTextStyle, true)
				if printedWidth > maxWidth {
					maxWidth = printedWidth
				}
				if end < len(item.MainText) {
					overflowing = true
				}

				// show selection hghlighted

				y++

				if y >= bottomLimit {
					break PARENT
				}
			}

		}

	}

	// We don't want the item text to get out of view. If the horizontal offset
	// is too high, we reset it and redraw. (That should be about as efficient
	// as calculating everything up front.)
	if l.horizontalOffset > 0 && maxWidth < width {
		l.horizontalOffset -= width - maxWidth
		l.Draw(screen)
	}
	l.overflowing = overflowing
}

// adjustOffset adjusts the vertical offset to keep the current selection in
// view.
func (l *DeepList) adjustOffset() {
	_, _, _, height := l.GetInnerRect()
	if height == 0 {
		return
	}
	currentItemOffset := getOffset(0, 0, l.currentItem, l.items)
	if currentItemOffset < l.itemOffset {
		l.itemOffset = currentItemOffset
	} else if l.showSecondaryText {
		if 2*(currentItemOffset-l.itemOffset) >= height-1 {
			l.itemOffset = (2*currentItemOffset + 3 - height) / 2
		}
	} else {
		if currentItemOffset-l.itemOffset >= height {
			l.itemOffset = currentItemOffset + 1 - height
		}
	}
}

// InputHandler returns the handler for this primitive.
func (l *DeepList) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return l.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		if event.Key() == tcell.KeyEscape {
			if l.done != nil {
				l.done()
			}
			return
		} else if len(l.items) == 0 {
			return
		}

		previousItem := l.currentItem

		switch key := event.Key(); key {
		case tcell.KeyTab, tcell.KeyDown:
			l.currentItem = moveSelectionIndex(0, l.currentItem, l.items, 1)
		case tcell.KeyBacktab, tcell.KeyUp:
			l.currentItem = moveSelectionIndex(0, l.currentItem, l.items, -1)
		case tcell.KeyRight:
			if l.overflowing {
				l.horizontalOffset += 2 // We shift by 2 to account for two-cell characters.
			} else {
				l.currentItem = moveSelectionIndex(0, l.currentItem, l.items, 1)
			}
		case tcell.KeyLeft:
			if l.horizontalOffset > 0 {
				l.horizontalOffset -= 2
			} else {
				l.currentItem = moveSelectionIndex(0, l.currentItem, l.items, -1)
			}
		case tcell.KeyHome:
			l.currentItem = []int{0}
		case tcell.KeyEnd:
			l.currentItem = []int{len(l.items) - 1}
		case tcell.KeyPgDn:
			_, _, _, height := l.GetInnerRect()
			l.currentItem = moveSelectionIndex(0, l.currentItem, l.items, height)
			//if l.currentItem >= len(l.items) {
			//	l.currentItem = len(l.items) - 1
			//}
		/*case tcell.KeyPgUp:
		_, _, _, height := l.GetInnerRect()
		l.currentItem -= height
		if l.currentItem < 0 {
			l.currentItem = 0
		}
		*/
		case tcell.KeyEnter:
			if l.currentItem[0] >= 0 && l.currentItem[0] < len(l.items) {
				item := l.items[l.currentItem[0]]
				if item.Selected != nil {
					item.Selected()
				}
				if l.selected != nil {
					l.selected(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
				}
			}
		case tcell.KeyRune:
			ch := event.Rune()
			if ch != ' ' {
				// It's not a space bar. Is it a shortcut?
				var found bool
				for index, item := range l.items {
					if item.Shortcut == ch {
						// We have a shortcut.
						found = true
						l.currentItem = []int{index}
						break
					}
				}
				if !found {
					break
				}
			}
			item := l.items[l.currentItem[0]]
			if item.Selected != nil {
				item.Selected()
			}
			if l.selected != nil {
				l.selected(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
			}
		}

		if l.currentItem[0] < 0 {
			if l.wrapAround {
				l.currentItem[0] = len(l.items) - 1
			} else {
				l.currentItem[0] = 0
			}
		} else if l.currentItem[0] >= len(l.items) {
			if l.wrapAround {
				l.currentItem[0] = 0
			} else {
				l.currentItem[0] = len(l.items) - 1
			}
		}

		if l.currentItem[0] != previousItem[0] && l.currentItem[0] < len(l.items) {
			if l.changed != nil {
				item := l.items[l.currentItem[0]]
				l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
			}
			l.adjustOffset()
		}
	})
}

// indexAtPoint returns the index of the list item found at the given position
// or a negative value if there is no such list item.
func (l *DeepList) indexAtPoint(x, y int) int {
	rectX, rectY, width, height := l.GetInnerRect()
	if rectX < 0 || rectX >= rectX+width || y < rectY || y >= rectY+height {
		return -1
	}

	index := y - rectY
	if l.showSecondaryText {
		index /= 2
	}
	index += l.itemOffset

	if index >= len(l.items) {
		return -1
	}
	return index
}

// MouseHandler returns the mouse handler for this primitive.
func (l *DeepList) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return l.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		if !l.InRect(event.Position()) {
			return false, nil
		}

		/*
			// Process mouse event.
			switch action {
			case MouseLeftClick:
				setFocus(l)
				index := l.indexAtPoint(event.Position())
				if index != -1 {
					item := l.items[index]
					if item.Selected != nil {
						item.Selected()
					}
					if l.selected != nil {
						l.selected(index, item.MainText, item.SecondaryText, item.Shortcut)
					}
					if index != l.currentItem {
						if l.changed != nil {
							l.changed(index, item.MainText, item.SecondaryText, item.Shortcut)
						}
						l.adjustOffset()
					}
					l.currentItem = index
				}
				consumed = true
			case MouseScrollUp:
				if l.itemOffset > 0 {
					l.itemOffset--
				}
				consumed = true
			case MouseScrollDown:
				lines := len(l.items) - l.itemOffset
				if l.showSecondaryText {
					lines *= 2
				}
				if _, _, _, height := l.GetInnerRect(); lines > height {
					l.itemOffset++
				}
				consumed = true
			}
		*/

		return
	})
}
