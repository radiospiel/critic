package main

import (
	"os"

	"git.15b.it/eno/critic/internal/cli"
	"git.15b.it/eno/critic/internal/tui"
)

func main() {
	if err := cli.Execute(tui.Run); err != nil {
		os.Exit(1)
	}
}
