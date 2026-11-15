package main

import "testing"

func TestWindowCommandsRequireARunningWindow(t *testing.T) {
	app := NewApp()
	commands := map[string]func() error{
		"minimize":          func() error { return app.MinimiseWindow("main") },
		"zoom":              func() error { return app.ZoomWindow("main") },
		"toggle fullscreen": func() error { return app.ToggleFullScreen("main") },
	}
	for name, command := range commands {
		if err := command(); err != errNoWindow {
			t.Fatalf("%s error = %v, want %v", name, err, errNoWindow)
		}
	}
}
