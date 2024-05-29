package main

import (
	"log/slog"
	"os"

	"github.com/ChristofferNissen/helmper/internal"
)

func main() {
	// invoke program and handle error
	err := internal.Program(os.Args[1:])
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
