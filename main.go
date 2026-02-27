package main

import (
	"fmt"
	"os"

	"github.com/jasonchiu/envlock/core/config"
	"github.com/jasonchiu/envlock/feature/cli"
)

func main() {
	config.LoadDotenvIfPresent()
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "envlock: %v\n", err)
		os.Exit(1)
	}
}
