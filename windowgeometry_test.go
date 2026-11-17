package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// screen builds a display for the placement tests. Only the fields the sizing
// rules read are set; Wails fills the rest on a real launch.
func screen(x, y, width, height int, primary bool) *application.Screen {
	return &application.Screen{
		IsPrimary: primary,
		WorkArea:  application.Rect{X: x, Y: y, Width: width, Height: height},
	}
}

func TestDefaultWindowGeometryFitsLargeScreen(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}
	got := defaultWindowGeometry(screens)

	want := windowGeometry{X: 360, Y: 152, Width: 1200, Height: 800}
	if got != want {
		t.Fatalf("defaultWindowGeometry() = %+v, want %+v", got, want)
	}
}

func TestDefaultWindowGeometryCapsToSmallScreen(t *testing.T) {
	// A 13" MacBook: 1280 points wide, about 695 below the menu bar and above
	// the Dock. The 1200x800 default must not hang under the Dock.
	screens := []*application.Screen{screen(0, 25, 1280, 695, true)}
	got := defaultWindowGeometry(screens)

	want := windowGeometry{X: 40, Y: 25, Width: 1200, Height: 695}
	if got != want {
		t.Fatalf("defaultWindowGeometry() = %+v, want %+v", got, want)
	}
}

func TestDefaultWindowGeometryWithoutScreens(t *testing.T) {
	got := defaultWindowGeometry(nil)
	want := windowGeometry{Width: 1200, Height: 800}
	if got != want {
		t.Fatalf("defaultWindowGeometry(nil) = %+v, want %+v", got, want)
	}
}

func TestResolveWindowGeometryRestoresVisibleFrame(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}
	saved := WindowFrame{X: 200, Y: 120, Width: 1000, Height: 700}

	got := resolveWindowGeometry(saved, screens)
	if got != (windowGeometry{X: 200, Y: 120, Width: 1000, Height: 700}) {
		t.Fatalf("resolveWindowGeometry() = %+v, want the saved frame", got)
	}
}

func TestResolveWindowGeometryRecentersLostWindow(t *testing.T) {
	// A frame from a monitor that is gone: x=4000 is off the remaining screen.
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}
	saved := WindowFrame{X: 4000, Y: 120, Width: 1000, Height: 700}

	got := resolveWindowGeometry(saved, screens)
	if want := defaultWindowGeometry(screens); got != want {
		t.Fatalf("resolveWindowGeometry() = %+v, want default %+v", got, want)
	}
}

func TestResolveWindowGeometryRejectsFrameUnderMenuBar(t *testing.T) {
	// The saved frame sits above the work area's top edge, so its title bar is
	// hidden by the menu bar and cannot be grabbed.
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}
	saved := WindowFrame{X: 200, Y: 0, Width: 1000, Height: 700}

	got := resolveWindowGeometry(saved, screens)
	if want := defaultWindowGeometry(screens); got != want {
		t.Fatalf("resolveWindowGeometry() = %+v, want default %+v", got, want)
	}
}

func TestResolveWindowGeometryShrinksFrameForNarrowerScreen(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1280, 695, true)}
	saved := WindowFrame{X: 100, Y: 100, Width: 1800, Height: 1000}

	got := resolveWindowGeometry(saved, screens)
	want := windowGeometry{X: 0, Y: 25, Width: 1280, Height: 695}
	if got != want {
		t.Fatalf("resolveWindowGeometry() = %+v, want %+v", got, want)
	}
}

func TestResolveWindowGeometryIgnoresEmptyFrame(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}

	got := resolveWindowGeometry(WindowFrame{}, screens)
	if want := defaultWindowGeometry(screens); got != want {
		t.Fatalf("resolveWindowGeometry() = %+v, want default %+v", got, want)
	}
}

func TestCascadeWindowGeometryOffsetsFrame(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}
	base := windowGeometry{X: 200, Y: 200, Width: 1000, Height: 700}

	got := cascadeWindowGeometry(base, screens)
	want := windowGeometry{X: 224, Y: 224, Width: 1000, Height: 700}
	if got != want {
		t.Fatalf("cascadeWindowGeometry() = %+v, want %+v", got, want)
	}
}

func TestCascadeWindowGeometryKeepsWindowOnScreen(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1280, 695, true)}
	base := windowGeometry{X: 900, Y: 500, Width: 1000, Height: 600}

	got := cascadeWindowGeometry(base, screens)
	if got.X+got.Width > 1280 || got.Y+got.Height > 720 {
		t.Fatalf("cascadeWindowGeometry() = %+v, want the frame inside the work area", got)
	}
	if got.X < 0 || got.Y < 25 {
		t.Fatalf("cascadeWindowGeometry() = %+v, want the frame inside the work area", got)
	}
}

func TestCascadeWindowGeometryFallsBackWithoutBase(t *testing.T) {
	screens := []*application.Screen{screen(0, 25, 1920, 1055, true)}

	got := cascadeWindowGeometry(windowGeometry{}, screens)
	if want := defaultWindowGeometry(screens); got != want {
		t.Fatalf("cascadeWindowGeometry() = %+v, want default %+v", got, want)
	}
}
