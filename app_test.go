package main

import (
	"context"
	"errors"
	"testing"
)

func TestStartupStoresContext(t *testing.T) {
	app := NewApp()

	ctx, cancel := context.WithCancel(context.Background())
	app.startup(ctx)
	cancel()

	// Runtime calls made after startup rely on the stored context; losing it
	// silently breaks the app, so assert it survives startup.
	if err := app.ctx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected stored context to be canceled after startup, got %v", err)
	}
}
