package main

import "context"

// App is the application struct. Exported methods become callable from the
// frontend through the generated Wails bindings.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so runtime
// methods can be called later.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
