package main

import (
	"os"

	"github.com/radiospiel/critic/src/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
