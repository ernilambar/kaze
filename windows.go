package main

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// ClosePrompt is the payload of app:confirm-close. The frontend titles the
// prompt from Reason ("close" or "quit") and answers through ResolveClose.
type ClosePrompt struct {
	Reason string `json:"reason"`
}

// windowSession is the Go-side state of one window. Tabs, dirty flags and
// pending launch paths live in that window's frontend, so the backend only
// remembers what a window has told it: whether it is listening, what it has not
// collected yet and whether it may close. A unit test registers a session
// without a Wails window, which is why window is nil-checked everywhere.
type windowSession struct {
	name   string
	window *application.WebviewWindow

	mu        sync.Mutex
	pending   LaunchRequest
	listening bool
	// reason is what the current close prompt is asking about.
	reason string

	unsaved        atomic.Bool
	confirmed      atomic.Bool
	closeRequested atomic.Bool

	// restoreGeometry marks the first window of a launch: it adopts the frame
	// saved in session.json, while later windows cascade from the first.
	// savedFrame is that stored frame, read once when the window is created.
	restoreGeometry bool
	savedFrame      WindowFrame
	// owner marks the one window whose tabs session.json restores and writes
	// on the next launch. Later windows park workspace tab sets but do not
	// replace the launch session with their own.
	owner bool
	// placed makes the move/size-once before the window is shown idempotent;
	// the runtime may report readiness more than once.
	placed atomic.Bool
}

// emit sends an event to this window. Wails stamps the sender's name, which the
// frontend uses to ignore commands aimed at another window.
func (s *windowSession) emit(name string, data ...any) {
	s.mu.Lock()
	window := s.window
	s.mu.Unlock()
	if window != nil {
		window.EmitEvent(name, data...)
	}
}

// show raises a window, restoring it first when it was minimised.
func (s *windowSession) show() {
	s.mu.Lock()
	window := s.window
	s.mu.Unlock()
	if window == nil {
		return
	}
	window.UnMinimise()
	window.Show()
}

// takeLaunch marks the window as listening and returns what it should open.
func (s *windowSession) takeLaunch() LaunchRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listening = true
	request := s.pending.normalized()
	s.pending = emptyLaunchRequest()
	return request
}

// queueLaunch records a request for a window that is still starting up, or
// returns it for delivery when the window is already listening.
func (s *windowSession) queueLaunch(request LaunchRequest) (LaunchRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending.Merge(request)
	if !s.listening {
		return LaunchRequest{}, false
	}
	payload := s.pending.normalized()
	s.pending = emptyLaunchRequest()
	return payload, true
}

// allowClose reports whether this window may go away without asking. The user
// confirming a prompt sets confirmed, which outlives the unsaved flag because
// "Don't Save" leaves the buffer dirty while the window closes.
func (s *windowSession) allowClose() bool {
	return !s.unsaved.Load() || s.confirmed.Load()
}

// registerWindow adds a session before its webview starts loading, so the
// frontend can collect its launch request as soon as it runs. The first window
// adopts any request queued before a window existed.
func (a *App) registerWindow(name string, request LaunchRequest) *windowSession {
	a.windowsMu.Lock()
	first := len(a.windows) == 0
	if first {
		request.RestoreSession = true
		if !a.pending.Empty() {
			request.Merge(a.pending)
			a.pending = emptyLaunchRequest()
		}
	}
	session := &windowSession{
		name:            name,
		pending:         request.normalized(),
		restoreGeometry: first,
		owner:           first,
	}
	a.windows[name] = session
	a.order = append(a.order, name)
	a.windowsMu.Unlock()
	return session
}

// session finds one window by name. An empty name means the caller could not
// read its own window name; the focused window is the best answer then.
func (a *App) session(name string) *windowSession {
	a.windowsMu.Lock()
	defer a.windowsMu.Unlock()

	if name != "" {
		return a.windows[name]
	}
	return a.focusedSessionLocked()
}

// focusedSession finds the window the user is looking at. Menu commands act on
// it because the native menu is shared by every window.
func (a *App) focusedSession() *windowSession {
	a.windowsMu.Lock()
	defer a.windowsMu.Unlock()
	return a.focusedSessionLocked()
}

func (a *App) focusedSessionLocked() *windowSession {
	if a.wails == nil {
		return nil
	}
	window := a.wails.Window.Current()
	if window == nil {
		return nil
	}
	return a.windows[window.Name()]
}

// primarySession is the first window opened, used when no window has focus.
func (a *App) primarySession() *windowSession {
	a.windowsMu.Lock()
	defer a.windowsMu.Unlock()
	return a.primarySessionLocked()
}

func (a *App) primarySessionLocked() *windowSession {
	for _, name := range a.order {
		if session, ok := a.windows[name]; ok {
			return session
		}
	}
	return nil
}

// targetSession is where an incoming launch request belongs: the focused window
// when there is one, otherwise the first.
func (a *App) targetSession() *windowSession {
	if session := a.focusedSession(); session != nil {
		return session
	}
	return a.primarySession()
}

// sessions returns the open windows in creation order.
func (a *App) sessions() []*windowSession {
	a.windowsMu.Lock()
	defer a.windowsMu.Unlock()

	list := make([]*windowSession, 0, len(a.order))
	for _, name := range a.order {
		if session, ok := a.windows[name]; ok {
			list = append(list, session)
		}
	}
	return list
}

// dirtySessions are the windows that still need an answer before the app may
// quit.
func (a *App) dirtySessions() []*windowSession {
	dirty := make([]*windowSession, 0, len(a.order))
	for _, session := range a.sessions() {
		if session.unsaved.Load() && !session.confirmed.Load() {
			dirty = append(dirty, session)
		}
	}
	return dirty
}

// nextWindowName numbers windows so a frontend can address its own session.
func (a *App) nextWindowName() string {
	number := a.windowSeq.Add(1)
	if number == 1 {
		return "main"
	}
	return fmt.Sprintf("window-%d", number)
}

// openWindow creates one editor window and its session. The session is
// registered before the webview starts, so the frontend's GetLaunchRequest can
// never race the request that created it.
func (a *App) openWindow(request LaunchRequest) error {
	if a.wails == nil {
		return errNoWindow
	}

	a.allowQuit.Store(false)
	name := a.nextWindowName()
	session := a.registerWindow(name, request)

	// The first window of a launch restores the saved frame; later windows
	// cascade from the window that spawned them. Both decisions need the
	// screens, which Wails only enumerates once the app is running, so the
	// window is created hidden and shown by placeWindow after the frontend is
	// alive. Showing it first would paint it at the fallback size and then
	// jump it to the real one.
	width, height := defaultWindowWidth, defaultWindowHeight
	if session.restoreGeometry {
		session.savedFrame = a.savedWindowGeometry()
		if session.savedFrame.Width >= minWindowWidth && session.savedFrame.Height >= minWindowHeight {
			width, height = session.savedFrame.Width, session.savedFrame.Height
		}
	}

	window := a.wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             name,
		Title:            a.meta.ProductName,
		Width:            width,
		Height:           height,
		MinWidth:         minWindowWidth,
		MinHeight:        minWindowHeight,
		Hidden:           true,
		BackgroundColour: application.NewRGB(30, 30, 30),
		URL:              "/",
		// Files dragged in from Finder arrive through AppKit, not the webview;
		// the frontend marks the whole window as the drop target and this
		// listener on each window receives the paths.
		EnableFileDrop:     true,
		UseApplicationMenu: true,
		// The frontend draws the window's only title bar. The native macOS bar
		// turns transparent with its title hidden and the web content extends
		// under it, so the traffic lights float over TitleBar.svelte instead of
		// stacking a second bar above it. The plain preset keeps the lights at
		// the standard height; the inset presets add a toolbar that pushes them
		// below the bar's midline.
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHidden,
		},
	})

	session.mu.Lock()
	session.window = window
	session.mu.Unlock()

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		a.windowClosing(session, event)
	})
	// The runtime reports ready once the page is loaded and its JavaScript is
	// running; only then do the screens and a stable window size exist, so the
	// window is sized, positioned and shown here.
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		a.placeWindow(session)
	})
	// Finder reports the paths to this window after the webview has confirmed
	// the point is on a drop target. Option is read from AppKit at drop time
	// because the drag never becomes a DOM event.
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		a.handleFileDrop(session, event.Context().DroppedFiles(), optionKeyHeld())
	})
	return nil
}

// placeWindow sizes and positions a window once its frontend is running, then
// shows it. The screens exist by then, so a restored frame can be checked
// against today's display layout before it is trusted.
func (a *App) placeWindow(session *windowSession) {
	session.mu.Lock()
	window := session.window
	session.mu.Unlock()
	if window == nil || a.wails == nil || !session.placed.CompareAndSwap(false, true) {
		return
	}

	// GetPrimary lazily enumerates the screens; GetAll alone would be empty on
	// the first call.
	if a.wails.Screen.GetPrimary() == nil {
		window.Show()
		return
	}
	screens := a.wails.Screen.GetAll()

	var frame windowGeometry
	if session.restoreGeometry {
		frame = resolveWindowGeometry(session.savedFrame, screens)
	} else {
		frame = cascadeWindowGeometry(a.cascadeOriginGeometry(), screens)
	}
	window.SetSize(frame.Width, frame.Height)
	window.SetPosition(frame.X, frame.Y)
	window.Show()

	// The zoom state is applied after the window is on screen: macOS ignores
	// zoom on a hidden window, and doing it now means the frame underneath is
	// the normal one, so turning the zoom off lands where the user left it.
	if session.restoreGeometry {
		if session.savedFrame.Fullscreen {
			window.Fullscreen()
		} else if session.savedFrame.Maximised {
			window.Maximise()
		}
	}
}

// cascadeOriginGeometry is the frame a New Window steps down and right from:
// the oldest open window's, unless it is zoomed, in which case the default is
// the only size that means anything.
func (a *App) cascadeOriginGeometry() windowGeometry {
	primary := a.primarySession()
	if primary == nil {
		return windowGeometry{}
	}
	primary.mu.Lock()
	window := primary.window
	primary.mu.Unlock()
	if window == nil || window.IsMaximised() || window.IsFullscreen() {
		return windowGeometry{}
	}
	width, height := window.Size()
	x, y := window.Position()
	return windowGeometry{X: x, Y: y, Width: width, Height: height}
}

// savedWindowGeometry reads the frame stored for the last launch. A missing or
// broken session file is not fatal: the caller falls back to the default.
func (a *App) savedWindowGeometry() WindowFrame {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state, err := a.loadSessionLocked()
	if err != nil {
		return WindowFrame{}
	}
	return state.Frame
}

// saveWindowGeometry records the frame a window is leaving with. Only the
// first open window of the launch owns the saved frame, so closing a later
// window does not replace where the user's main window was.
func (a *App) saveWindowGeometry(session *windowSession) {
	if session == nil || a.wails == nil {
		return
	}
	session.mu.Lock()
	window := session.window
	session.mu.Unlock()
	if window == nil {
		return
	}
	if primary := a.primarySession(); primary != nil && primary != session {
		return
	}

	saved := WindowFrame{
		Maximised:  window.IsMaximised(),
		Fullscreen: window.IsFullscreen(),
	}
	// A zoomed window's size is the screen's, not the frame the user sized by
	// hand; keep the stored one so un-zooming after a relaunch restores it.
	if !saved.Maximised && !saved.Fullscreen {
		saved.Width, saved.Height = window.Size()
		saved.X, saved.Y = window.Position()
	}
	a.storeWindowGeometry(saved)
}

// storeWindowGeometry writes the frame into session.json. A zoomed window keeps
// the normal size and position already stored, so the only fields that change
// are the state flags. Geometry loss is cosmetic, so a failed write is not
// reported to the user.
func (a *App) storeWindowGeometry(frame WindowFrame) {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state, err := a.loadSessionLocked()
	if err != nil {
		return
	}
	next := state.Frame
	if frame.Width > 0 && frame.Height > 0 {
		next.Width = frame.Width
		next.Height = frame.Height
		next.X = frame.X
		next.Y = frame.Y
	}
	next.Maximised = frame.Maximised
	next.Fullscreen = frame.Fullscreen
	state.Frame = normalizeWindowFrame(next)
	a.sessionStore = state
	_ = writeSessionFile(state)
}

// windowClosing gives a window the same unsaved-buffer guard as quitting. A
// cancelled event keeps it open and hands the decision to its frontend.
func (a *App) windowClosing(session *windowSession, event *application.WindowEvent) {
	// Record the frame before the window goes away. A cancelled close leaves
	// it saved at its current place, which is where it still is.
	a.saveWindowGeometry(session)
	if session.allowClose() {
		a.forgetWindow(session)
		return
	}
	event.Cancel()
	a.requestClose(session, "close")
}

// forgetWindow drops the session of a window that is going away. Its watcher
// registrations go with it, so a closed window cannot receive change events.
func (a *App) forgetWindow(session *windowSession) {
	a.windowsMu.Lock()
	delete(a.windows, session.name)
	for index, name := range a.order {
		if name == session.name {
			a.order = append(a.order[:index], a.order[index+1:]...)
			break
		}
	}
	a.windowsMu.Unlock()
	a.watcher.forget(session.name)
}

// requestClose asks one window's frontend to confirm before unsaved edits are
// lost. Only one prompt per window is ever on screen.
func (a *App) requestClose(session *windowSession, reason string) {
	if session == nil || !session.closeRequested.CompareAndSwap(false, true) {
		return
	}
	session.mu.Lock()
	session.reason = reason
	session.mu.Unlock()
	session.emit(closeRequestEvent, ClosePrompt{Reason: reason})
}

// SetUnsavedChanges records whether one window has edits that are not on disk
// yet. Each window reports its own state, so closing one never asks about
// another window's buffers.
func (a *App) SetUnsavedChanges(window string, unsaved bool) {
	if session := a.session(window); session != nil {
		session.unsaved.Store(unsaved)
	}
}

// shouldQuit answers macOS's terminate request. Any dirty window cancels the
// quit; the windows are asked one at a time, and the last answer terminates.
func (a *App) shouldQuit() bool {
	if a.allowQuit.Load() {
		a.savePrimaryWindowGeometry()
		return true
	}
	if len(a.dirtySessions()) == 0 {
		a.savePrimaryWindowGeometry()
		return true
	}
	a.quitInProgress.Store(true)
	a.askNextQuit()
	return false
}

// savePrimaryWindowGeometry records the frame of the window that owns the
// saved geometry. Closing a window saves it too, but a quit can terminate the
// process without a WindowClosing hook for the last window.
func (a *App) savePrimaryWindowGeometry() {
	if primary := a.primarySession(); primary != nil {
		a.saveWindowGeometry(primary)
	}
}

// askNextQuit prompts the next dirty window. A prompt already on screen blocks
// the next one, so Cmd+Q never stacks dialogs.
func (a *App) askNextQuit() {
	if !a.quitInProgress.Load() {
		return
	}
	for _, session := range a.dirtySessions() {
		if session.closeRequested.Load() {
			return
		}
		a.requestClose(session, "quit")
		return
	}
	a.quitInProgress.Store(false)
	a.allowQuit.Store(true)
	a.quit()
}

// ResolveClose answers a close request for one window. Cancelling clears the
// request so the next attempt asks again; confirming either closes that window
// or continues the app quit with the next dirty window.
func (a *App) ResolveClose(window string, quit bool) {
	session := a.session(window)
	if session == nil {
		return
	}

	session.mu.Lock()
	reason := session.reason
	session.reason = ""
	session.mu.Unlock()
	session.closeRequested.Store(false)

	if !quit {
		if reason == "quit" {
			// A cancel abandons the whole quit, including windows that already
			// answered it; the next Cmd+Q must ask all of them again.
			a.quitInProgress.Store(false)
			for _, session := range a.sessions() {
				session.confirmed.Store(false)
			}
		}
		return
	}

	switch reason {
	case "close":
		session.confirmed.Store(true)
		// Closing the last window ends the app. Let that termination through
		// without a second prompt even when the user chose "Don't Save".
		if len(a.sessions()) == 1 {
			a.allowQuit.Store(true)
		}
		if session.window != nil {
			session.window.Close()
		}
	case "quit":
		session.confirmed.Store(true)
		a.askNextQuit()
	}
}

// quit shuts the app down so its shutdown path runs.
func (a *App) quit() {
	if a.wails != nil {
		a.savePrimaryWindowGeometry()
		a.wails.Quit()
	}
}

// NewWindow opens an empty editor window. It shares the saved sidebar roots and
// settings, but its tabs are its own.
func (a *App) NewWindow() error {
	return a.openWindow(emptyLaunchRequest())
}

// CloseWindow closes one window through the same unsaved-buffer guard as its
// red close button.
func (a *App) CloseWindow(window string) error {
	session := a.session(window)
	if session == nil || session.window == nil {
		return errNoWindow
	}
	session.window.Close()
	return nil
}

// OpenFolderInNewWindow asks for a folder and opens it as the only root of a
// new window. The roots are saved, so the next launch starts there too.
func (a *App) OpenFolderInNewWindow() error {
	if a.wails == nil {
		return errNoWindow
	}
	path, err := a.wails.Dialog.OpenFile().
		SetTitle("Open Folder in New Window").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		PromptForSingleSelection()
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}
	return a.openWindow(LaunchRequest{Folders: []string{path}, ReplaceFolders: true})
}

// MoveTabToNewWindow hands one buffer to a new window. The frontend removes the
// tab from its own list once this returns, because the buffer now lives in the
// new window.
func (a *App) MoveTabToNewWindow(buffer LaunchBuffer) error {
	return a.openWindow(LaunchRequest{Buffers: []LaunchBuffer{buffer}})
}

// BringAllToFront raises every window, restoring minimised ones.
func (a *App) BringAllToFront() {
	if a.wails == nil {
		return
	}
	for _, window := range a.wails.Window.GetAll() {
		window.UnMinimise()
		window.Show()
	}
}
