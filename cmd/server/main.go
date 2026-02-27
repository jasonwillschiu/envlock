package main

import (
	"fmt"
	"net/http"
	"os"

	coreauth "github.com/jasonchiu/envlock/core/auth"
	coreconfig "github.com/jasonchiu/envlock/core/config"
	corerouter "github.com/jasonchiu/envlock/core/router"
)

func main() {
	coreconfig.LoadDotenvIfPresent()

	cfg := coreconfig.Load()
	store := coreauth.NewMemoryStore()
	handler := corerouter.New(corerouter.Deps{
		Config:        cfg,
		CLILoginStore: store,
	})

	fmt.Printf("envlock server listening on %s\n", cfg.Addr)
	fmt.Printf("base url: %s\n", cfg.BaseURL)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "envlock-server: %v\n", err)
		os.Exit(1)
	}
}
