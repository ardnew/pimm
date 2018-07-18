package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	DrawCycleDuration    = 5 * time.Millisecond
	PlaylistIndexInvalid = -1
)

const (
	PageIDBrowser   = "browse"
	PageIDQuitModal = "quit"

	LibraryFocusRune     = 'l'
	MediaFocusRune       = 'm'
	MediaDetailFocusRune = 'i'
	PlaylistFocusRune    = 'p'
	LogFocusRune         = 'v'
	QuitRune             = 'q'
)

type UIView interface {
	UI() *UI
	FocusRune() rune
	Obscura() *tview.Flex
	Proportion() int
	Visible() bool
	SetVisible(bool)
	LockFocus(bool)
}

type UI struct {
	app *tview.Application

	sigDraw chan interface{}

	media map[string]*MediaInfo

	focusLocked      bool
	focusLockedView  UIView
	focusTitleColor  map[bool]tcell.Color
	focusBorderColor map[bool]tcell.Color

	libraryView     *LibraryView
	mediaView       *MediaView
	mediaDetailView *MediaDetailView
	playlistView    *PlaylistView
	logView         *LogView
	pageControl     *PageControl
}

type PageControl struct {
	*tview.Pages
	focusedView tview.Primitive
}

type MediaInfo struct {
	library       *Library
	media         *Media
	libraryNode   *tview.TreeNode
	mediaCell     *tview.TableCell
	playlistIndex int
}

func NewMediaInfo(library *Library, media *Media) *MediaInfo {
	return &MediaInfo{
		library:       library,
		media:         media,
		libraryNode:   nil,
		mediaCell:     nil,
		playlistIndex: PlaylistIndexInvalid,
	}
}

func (ref *MediaInfo) String() string {
	return fmt.Sprintf("%s: %s", ref.library, ref.media)
}

type ViewPrimitive struct {
	view UIView
	prim tview.Primitive
}

var (
	focusKeyView = map[rune]ViewPrimitive{}
)

func NewUI(opt *Options) *UI {

	self := tview.NewApplication()
	sigDraw := make(chan interface{})

	//
	// containers for the primitive views
	//
	mediaRowsLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	browseColsLayout := tview.NewFlex().SetDirection(tview.FlexColumn)
	browseMainLayout := tview.NewFlex().SetDirection(tview.FlexRow)

	//
	// construct the primitive views
	//
	libraryView := NewLibraryView(browseColsLayout)
	mediaView := NewMediaView(mediaRowsLayout)
	mediaDetailView := NewMediaDetailView(mediaRowsLayout)
	playlistView := NewPlaylistView(browseColsLayout)
	logView := NewLogView(browseMainLayout)

	//
	// composition of container views
	//
	mediaRowsLayout.AddItem(mediaView, 0, mediaView.Proportion(), false)
	mediaRowsLayout.AddItem(mediaDetailView, 0, mediaDetailView.Proportion(), false)

	browseColsLayout.AddItem(libraryView, 0, libraryView.Proportion(), false)
	browseColsLayout.AddItem(mediaRowsLayout, 0, 2, false)
	browseColsLayout.AddItem(playlistView, 0, playlistView.Proportion(), false)

	browseMainLayout.AddItem(browseColsLayout, 0, 3, false)
	browseMainLayout.AddItem(logView, 0, logView.Proportion(), false)

	//
	// root-level container selects which app page to display
	//
	pageControl := &PageControl{
		Pages:       tview.NewPages(),
		focusedView: mediaView,
	}
	pageControl.AddPage(PageIDBrowser, browseMainLayout, true, true)

	//
	// construct reference table for focus key handlers
	//
	focusKeyView[libraryView.FocusRune()] = ViewPrimitive{libraryView, libraryView}
	focusKeyView[mediaView.FocusRune()] = ViewPrimitive{mediaView, mediaView}
	focusKeyView[mediaDetailView.FocusRune()] = ViewPrimitive{mediaDetailView, mediaDetailView}
	focusKeyView[playlistView.FocusRune()] = ViewPrimitive{playlistView, playlistView}
	focusKeyView[logView.FocusRune()] = ViewPrimitive{logView, logView}

	//
	// layout definition of entire UI
	//
	ui := &UI{

		app: self,

		sigDraw: sigDraw,

		media: make(map[string]*MediaInfo),

		focusLocked:     false,
		focusLockedView: nil,
		focusTitleColor: map[bool]tcell.Color{
			false: tcell.ColorWhite,
			true:  tcell.ColorDeepSkyBlue,
		},
		focusBorderColor: map[bool]tcell.Color{
			false: tcell.ColorWhite,
			true:  tcell.ColorSteelBlue,
		},

		libraryView:     libraryView,
		mediaView:       mediaView,
		mediaDetailView: mediaDetailView,
		playlistView:    playlistView,
		logView:         logView,
		pageControl:     pageControl,
	}

	//
	// initially focused view
	//
	self.SetRoot(pageControl, true)
	self.SetFocus(mediaView)
	mediaView.obscura.SetTitleColor(ui.focusTitleColor[true])
	mediaView.obscura.SetBorderColor(ui.focusBorderColor[true])

	//
	// backreference so each view has easy access to every other
	//
	ui.libraryView.ui = ui
	ui.mediaView.ui = ui
	ui.mediaDetailView.ui = ui
	ui.playlistView.ui = ui
	ui.logView.ui = ui

	go ui.DrawCycle()

	return ui
}

func (ui *UI) DrawCycle() {

	cycle := time.Tick(DrawCycleDuration)

	for {
		select {
		case <-cycle: // sufficient time has elapsed

			select {
			case <-ui.sigDraw: // we have an update available

				// the call to Draw() should be the -final- operation performed
				// during the cycle to prevent a race condition that would cause
				// updates to be drained without ever being handled by Draw().
				// -------------------------------------------------------------
				// e.g.: <<  1.) cycle start, 2.) Draw(), 3.) new sigDraw event,
				//           4.) drain sigDraw, 5.) cycle end  >>
				//       in this scenario, the new sigDraw event (3.) will never
				//       be handled by Draw() and will depend on some subsequent
				//       event to occur before the draw update ever happens
				// -------------------------------------------------------------
				for empty := false; !empty; {
					select {
					case <-ui.sigDraw:
					default:
						empty = true
					}
				}
				ui.app.Draw()
			}
		}
	}
}

type ModalView struct {
	*tview.Modal
}

func isQuitEventKeyRune(keyRune rune) bool { return QuitRune == keyRune }
func (ui *UI) ConfirmQuitPrompt() {

	modalView := &ModalView{Modal: tview.NewModal()}
	modalView.SetText("Are you a quitter?")
	modalView.AddButtons([]string{"Yes!", " No "})
	modalView.SetDoneFunc(
		func(buttonIndex int, buttonLabel string) {
			switch {
			case 0 == buttonIndex: // "Quit"
				ui.app.Stop()
			case 1 == buttonIndex || buttonIndex < 0: // "Cancel"/ESC pressed
				ui.pageControl.RemovePage(PageIDQuitModal)
				ui.app.SetFocus(ui.pageControl.focusedView)
			}
		})

	ui.pageControl.AddPage(PageIDQuitModal, modalView, true, true)
}

func (ui *UI) GlobalInputHandled(
	view UIView, event *tcell.EventKey, setFocus func(p tview.Primitive)) bool {

	inKey := event.Key()
	inMask := event.Modifiers()
	inRune := event.Rune()

	switch inMask {

	case tcell.ModShift:
	case tcell.ModCtrl:

		switch inKey {

		case tcell.KeyCtrlC:
			ui.ConfirmQuitPrompt()
			return true
		}

	case tcell.ModAlt:

		switch target, ok := focusKeyView[inRune]; {

		case ok:
			target.view.SetVisible(!target.view.Visible())

		}

	case tcell.ModMeta:
	case tcell.ModNone:

		switch inKey {

		case tcell.KeyEscape:
			focusedView := ui.pageControl.focusedView.(UIView)
			if focusedView.Visible() {
				focusedView.LockFocus(!ui.focusLocked)
			}

		case tcell.KeyRune:

			switch {
			case isQuitEventKeyRune(inRune):
				ui.ConfirmQuitPrompt()
				return true

			default:
				if target, ok := focusKeyView[inRune]; ok {
					if !ui.focusLocked {
						infoLog.Logf("focused: %T", target.prim)
						ui.pageControl.focusedView = target.prim
						setFocus(target.prim)
						target.view.SetVisible(true)
						return true
					}
				}
			}

		default:
			// nothing
		}

	}

	return false
}

// -----------------------------------------------------------------------------
//	(pimm) LibraryView
// -----------------------------------------------------------------------------

func (ui *UI) AddLibrary(library *Library) {
	ui.libraryView.AddLibrary(library)
	ui.sigDraw <- ui.libraryView
}
func (ui *UI) AddLibraryDirectory(library *Library, dir string) {
	ui.libraryView.AddLibraryDirectory(library, dir)
	ui.sigDraw <- ui.libraryView
}
func (ui *UI) AddMedia(library *Library, media *Media) {
	ui.libraryView.AddMedia(library, media)
	ui.sigDraw <- ui.libraryView
}
