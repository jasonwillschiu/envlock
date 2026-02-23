package main

import (
	"fmt"
	"os"

	"github.com/jasonchiu/envlock/internal/app"
	"github.com/jasonchiu/envlock/internal/config"
)

func main() {
	config.LoadDotenvIfPresent()
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "envlock: %v\n", err)
		os.Exit(1)
	}
}
