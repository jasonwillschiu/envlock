package cliauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	coreauth "github.com/jasonchiu/envlock/core/auth"
	coreconfig "github.com/jasonchiu/envlock/core/config"
)

type Handler struct {
	Config coreconfig.Runtime
	Store  *coreauth.MemoryStore
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/cli/login/start", h.start)
	r.Get("/login/cli/authorize", h.authorizePage)
	r.Post("/login/cli/authorize", h.authorizePage)
	r.Post("/api/cli/login/exchange", h.exchange)
	r.Get("/api/cli/whoami", h.whoami)
}

type startRequest struct {
	CallbackURL string `json:"callback_url,omitempty"`
}

type startResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		httpErrorJSON(w, http.StatusBadRequest, "invalid json body")
		return
	}
	p, err := h.Store.StartCLILogin(strings.TrimSpace(req.CallbackURL), h.Config.CLILoginCodeTTL)
	if err != nil {
		httpErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	authURL := h.Config.BaseURL + "/login/cli/authorize?state=" + url.QueryEscape(p.State)
	writeJSON(w, http.StatusOK, startResponse{AuthURL: authURL, State: p.State})
}

func (h *Handler) authorizePage(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}
	p, err := h.Store.GetPendingCLILogin(state, time.Now().UTC())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := coreauth.User{
		ID:          "dev:" + strings.ToLower(strings.TrimSpace(h.Config.DevUserEmail)),
		Email:       strings.TrimSpace(h.Config.DevUserEmail),
		DisplayName: strings.TrimSpace(h.Config.DevUserDisplay),
	}
	if user.Email == "" {
		user.Email = "dev@example.com"
	}
	if user.ID == "dev:" {
		user.ID = "dev:user"
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Email
	}

	if r.Method == http.MethodPost || h.Config.DevAutoApproveCLI {
		code, err := h.Store.IssueCodeForState(state, user, h.Config.CLILoginCodeTTL, time.Now().UTC())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p.CallbackURL != "" {
			cb, err := url.Parse(p.CallbackURL)
			if err == nil {
				q := cb.Query()
				q.Set("code", code.Code)
				q.Set("state", state)
				cb.RawQuery = q.Encode()
				http.Redirect(w, r, cb.String(), http.StatusSeeOther)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<html><body><h1>envlock dev login</h1><p>Copy this code into the CLI:</p><pre>%s</pre></body></html>`, code.Code)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<html><body><h1>Authorize envlock CLI login</h1><p>User: %s</p><form method="post"><button type="submit">Approve</button></form></body></html>`, user.Email)
}

type exchangeRequest struct {
	Code  string `json:"code"`
	State string `json:"state,omitempty"`
}

type exchangeResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time     `json:"expires_at,omitempty"`
	User         coreauth.User `json:"user"`
}

func (h *Handler) exchange(w http.ResponseWriter, r *http.Request) {
	var req exchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErrorJSON(w, http.StatusBadRequest, "invalid json body")
		return
	}
	token, err := h.Store.ExchangeCode(
		strings.TrimSpace(req.Code),
		strings.TrimSpace(req.State),
		h.Config.AccessTokenTTL,
		h.Config.RefreshTokenTTL,
		time.Now().UTC(),
	)
	if err != nil {
		httpErrorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, exchangeResponse{
		AccessToken:  token.Token,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
		User:         token.User,
	})
}

func (h *Handler) whoami(w http.ResponseWriter, r *http.Request) {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(authz, prefix) {
		httpErrorJSON(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(authz, prefix))
	user, err := h.Store.ValidateAccessToken(token, time.Now().UTC())
	if err != nil {
		httpErrorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpErrorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": strings.TrimSpace(msg)})
}
