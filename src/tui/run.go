package tui

import (
	"github.org/radiospiel/critic/teapot"
)

func Run(args *Args) error {
	// Create delegate (critic-specific logic)
	delegate := NewDelegate(args)

	// Create and run the application using teapot.App
	app := teapot.NewApp(delegate.mainLayout, delegate)
	delegate.app = app // Give delegate access to app for focus manager

	return app.Run()
}
