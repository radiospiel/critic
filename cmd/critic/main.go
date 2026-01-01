package main

import (
	"os"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/cli"
)

func main() {
	if err := cli.Execute(app.Run); err != nil {
		os.Exit(1)
	}
}
