package main

import "testing"

func TestWindowCommandsRequireARunningWindow(t *testing.T) {
	app := NewApp()
	commands := map[string]func() error{
		"minimize":          app.MinimiseWindow,
		"zoom":              app.ZoomWindow,
		"toggle fullscreen": app.ToggleFullScreen,
	}
	for name, command := range commands {
		if err := command(); err != errNoWindow {
			t.Fatalf("%s error = %v, want %v", name, err, errNoWindow)
		}
	}
}
