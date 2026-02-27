package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	coreauth "github.com/jasonchiu/envlock/core/auth"
	coreconfig "github.com/jasonchiu/envlock/core/config"
	"github.com/jasonchiu/envlock/feature/cliauth"
)

type Deps struct {
	Config        coreconfig.Runtime
	CLILoginStore *coreauth.MemoryStore
}

func New(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Logger)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	h := &cliauth.Handler{
		Config: deps.Config,
		Store:  deps.CLILoginStore,
	}
	h.RegisterRoutes(r)

	return r
}
