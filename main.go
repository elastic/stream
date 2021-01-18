package main

import (
	"os"

	"github.com/andrewkroh/stream/command"
)

func main() {
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
