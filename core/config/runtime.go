package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Runtime struct {
	Addr              string
	BaseURL           string
	DevUserEmail      string
	DevUserDisplay    string
	CLILoginCodeTTL   time.Duration
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	DevAutoApproveCLI bool
}

func Load() Runtime {
	addr := strings.TrimSpace(os.Getenv("ENVLOCK_SERVER_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("ENVLOCK_SERVER_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://" + addr
	}

	return Runtime{
		Addr:              addr,
		BaseURL:           baseURL,
		DevUserEmail:      envOrDefault("ENVLOCK_SERVER_DEV_USER_EMAIL", "dev@example.com"),
		DevUserDisplay:    envOrDefault("ENVLOCK_SERVER_DEV_USER_NAME", "Envlock Dev User"),
		CLILoginCodeTTL:   durationOrDefault("ENVLOCK_SERVER_CLI_CODE_TTL_SEC", 300),
		AccessTokenTTL:    durationOrDefault("ENVLOCK_SERVER_ACCESS_TTL_SEC", 3600),
		RefreshTokenTTL:   durationOrDefault("ENVLOCK_SERVER_REFRESH_TTL_SEC", 86400),
		DevAutoApproveCLI: boolOrDefault("ENVLOCK_SERVER_DEV_AUTO_APPROVE_CLI_LOGIN", true),
	}
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func durationOrDefault(envKey string, sec int) time.Duration {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return time.Duration(sec) * time.Second
}

func boolOrDefault(envKey string, fallback bool) bool {
	if v := strings.TrimSpace(strings.ToLower(os.Getenv(envKey))); v != "" {
		switch v {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}
