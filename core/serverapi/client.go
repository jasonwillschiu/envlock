package serverapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) (*Client, error) {
	u := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if u == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	return &Client{
		baseURL: u,
		http:    &http.Client{Timeout: 20 * time.Second},
	}, nil
}

type CLILoginStartRequest struct {
	CallbackURL string `json:"callback_url,omitempty"`
}

type CLILoginStartResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

type CLILoginExchangeRequest struct {
	Code  string `json:"code"`
	State string `json:"state,omitempty"`
}

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type CLILoginExchangeResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	User         User      `json:"user"`
}

func (c *Client) StartCLILogin(ctx context.Context, req CLILoginStartRequest) (CLILoginStartResponse, error) {
	var out CLILoginStartResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/cli/login/start", "", req, &out); err != nil {
		return CLILoginStartResponse{}, err
	}
	return out, nil
}

func (c *Client) ExchangeCLILogin(ctx context.Context, req CLILoginExchangeRequest) (CLILoginExchangeResponse, error) {
	var out CLILoginExchangeResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/cli/login/exchange", "", req, &out); err != nil {
		return CLILoginExchangeResponse{}, err
	}
	return out, nil
}

func (c *Client) WhoAmI(ctx context.Context, accessToken string) (User, error) {
	var out User
	if err := c.doJSON(ctx, http.MethodGet, "/api/cli/whoami", accessToken, nil, &out); err != nil {
		return User{}, err
	}
	return out, nil
}

func (c *Client) doJSON(ctx context.Context, method, path, accessToken string, reqBody any, dst any) error {
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		text := strings.TrimSpace(string(msg))
		if text == "" {
			text = resp.Status
		}
		return fmt.Errorf("server %s %s: %s", method, path, text)
	}
	if dst == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}
