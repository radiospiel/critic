package main

import (
	"os"

	"git.15b.it/eno/critic/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
