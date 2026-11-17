package main

import "github.com/wailsapp/wails/v3/pkg/application"

// Window sizing rules. A new window wants 1200x800, but never more than the
// screen can show; a restored window keeps its saved frame only while a usable
// piece of its title bar still lands on a screen.
const (
	defaultWindowWidth  = 1200
	defaultWindowHeight = 800
	minWindowWidth      = 600
	minWindowHeight     = 400

	// windowCascadeOffset steps a New Window down and right from the window
	// that spawned it, so it does not hide the original exactly.
	windowCascadeOffset = 24

	// A restored frame must leave at least this much of its title bar on a
	// screen. A window 2px on-screen is technically visible and practically
	// lost; this is enough to grab and drag back.
	titleBarVisibleWidth  = 120
	titleBarVisibleHeight = 48
)

// WindowFrame is the saved frame of the main window, held in session.json.
// An empty struct means no frame has been saved yet, so the next launch starts
// from the screen-sized default. The size and position are in DIPs, the units
// Wails reports.
//
// A maximised or fullscreen window stores the state here but keeps the normal
// frame in width/height/x/y, so turning the zoom off after a relaunch returns
// the window the user last sized by hand.
type WindowFrame struct {
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	Maximised  bool `json:"maximised"`
	Fullscreen bool `json:"fullscreen"`
}

// normalizeWindowFrame drops the size and position of a frame too small to hold
// the window, or one whose size was cleared by hand, so the next launch falls
// back to the screen-sized default. The zoom flags survive: a window that was
// only ever used maximised has no normal frame, yet must reopen maximised.
func normalizeWindowFrame(frame WindowFrame) WindowFrame {
	if frame.Width < minWindowWidth || frame.Height < minWindowHeight {
		return WindowFrame{Maximised: frame.Maximised, Fullscreen: frame.Fullscreen}
	}
	return frame
}

// windowGeometry is a resolved frame in DIPs, in the Y-down screen space Wails
// reports: X and Y are the window's top-left corner, matching Screen.WorkArea.
type windowGeometry struct {
	X      int
	Y      int
	Width  int
	Height int
}

// resolveWindowGeometry decides where the first window of a launch goes. A
// saved frame is used only when its title bar is still reachable; otherwise the
// frame was left on a display that changed or went away, and the window starts
// centred at the screen-sized default instead.
func resolveWindowGeometry(saved WindowFrame, screens []*application.Screen) windowGeometry {
	if saved.Width >= minWindowWidth && saved.Height >= minWindowHeight {
		frame := windowGeometry{
			X:      saved.X,
			Y:      saved.Y,
			Width:  saved.Width,
			Height: saved.Height,
		}
		if screen := screenHoldingTitleBar(frame, screens); screen != nil {
			return fitIntoWorkArea(frame, screen.WorkArea)
		}
	}
	return defaultWindowGeometry(screens)
}

// defaultWindowGeometry centres the default size on the primary screen, capped
// to the space the screen actually shows: a 1280x800 laptop has about 720
// points below the menu bar, so an 800-point window would hang under the Dock.
func defaultWindowGeometry(screens []*application.Screen) windowGeometry {
	frame := windowGeometry{Width: defaultWindowWidth, Height: defaultWindowHeight}
	screen := primaryScreen(screens)
	if screen == nil || screen.WorkArea.Width <= 0 || screen.WorkArea.Height <= 0 {
		return frame
	}
	work := screen.WorkArea
	frame.Width = minInt(defaultWindowWidth, work.Width)
	frame.Height = minInt(defaultWindowHeight, work.Height)
	frame.X = work.X + (work.Width-frame.Width)/2
	frame.Y = work.Y + (work.Height-frame.Height)/2
	return frame
}

// cascadeWindowGeometry offsets a frame from the window that spawned it. Each
// new window lands a little further down and right, but never off-screen; when
// the offset would push it out of reach it starts centred instead.
func cascadeWindowGeometry(base windowGeometry, screens []*application.Screen) windowGeometry {
	if len(screens) == 0 {
		return defaultWindowGeometry(nil)
	}
	if base.Width < minWindowWidth || base.Height < minWindowHeight {
		return defaultWindowGeometry(screens)
	}

	base.X += windowCascadeOffset
	base.Y += windowCascadeOffset
	if screen := screenHoldingTitleBar(base, screens); screen != nil {
		return fitIntoWorkArea(base, screen.WorkArea)
	}
	return defaultWindowGeometry(screens)
}

// fitIntoWorkArea keeps a window fully on one screen after its size was capped
// to that screen. Capping alone is not enough: a frame saved on a wide display
// and reopened on a narrow one would otherwise hang over the right edge.
func fitIntoWorkArea(frame windowGeometry, work application.Rect) windowGeometry {
	frame.Width = fitLength(frame.Width, minWindowWidth, work.Width)
	frame.Height = fitLength(frame.Height, minWindowHeight, work.Height)
	frame.X = clampInt(frame.X, work.X, work.X+work.Width-frame.Width)
	frame.Y = clampInt(frame.Y, work.Y, work.Y+work.Height-frame.Height)
	return frame
}

// fitLength keeps length within [minimum, available]. A work area smaller than
// the minimum is the one case where the minimum loses: the window should fill
// the screen rather than overflow it.
func fitLength(length, minimum, available int) int {
	if available < minimum {
		return maxInt(available, 1)
	}
	return clampInt(length, minimum, available)
}

// screenHoldingTitleBar returns the screen whose work area shows enough of the
// frame's title bar to grab, preferring the one showing the most. The title bar
// strip is the frame's first titleBarVisibleHeight points.
func screenHoldingTitleBar(frame windowGeometry, screens []*application.Screen) *application.Screen {
	var best *application.Screen
	bestArea := 0
	for _, screen := range screens {
		work := screen.WorkArea
		left := maxInt(frame.X, work.X)
		right := minInt(frame.X+frame.Width, work.X+work.Width)
		top := maxInt(frame.Y, work.Y)
		bottom := minInt(frame.Y+titleBarVisibleHeight, work.Y+work.Height)
		if right-left < titleBarVisibleWidth || bottom-top < titleBarVisibleHeight {
			continue
		}
		if area := (right - left) * (bottom - top); area > bestArea {
			best, bestArea = screen, area
		}
	}
	return best
}

// primaryScreen prefers the display macOS calls primary and falls back to the
// first attached screen, which is enough to place a window.
func primaryScreen(screens []*application.Screen) *application.Screen {
	for _, screen := range screens {
		if screen.IsPrimary {
			return screen
		}
	}
	if len(screens) > 0 {
		return screens[0]
	}
	return nil
}

func clampInt(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
