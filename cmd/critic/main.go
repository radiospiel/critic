package main

import (
	"os"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/cli"
)

func main() {
	// Set the command handler
	cli.OnCommand(app.Run)

	// Execute the CLI
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
