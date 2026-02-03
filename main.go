//go:build !gui
// +build !gui

package main

import (
	"os"

	"noisyzip/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
