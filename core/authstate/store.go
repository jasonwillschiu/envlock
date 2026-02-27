package authstate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var ErrNotFound = errors.New("auth state not found")

type User struct {
	ID          string `toml:"id,omitempty"`
	Email       string `toml:"email,omitempty"`
	DisplayName string `toml:"display_name,omitempty"`
}

type State struct {
	Version      int       `toml:"version"`
	ServerURL    string    `toml:"server_url"`
	AccessToken  string    `toml:"access_token,omitempty"`
	RefreshToken string    `toml:"refresh_token,omitempty"`
	ExpiresAt    time.Time `toml:"expires_at,omitempty"`
	User         User      `toml:"user"`
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "envlock"), nil
}

func DefaultPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.toml"), nil
}

func Load(path string) (State, error) {
	var s State
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return State{}, ErrNotFound
		}
		return State{}, err
	}
	if _, err := toml.DecodeFile(path, &s); err != nil {
		return State{}, err
	}
	if s.Version == 0 {
		s.Version = 1
	}
	s.ServerURL = strings.TrimRight(strings.TrimSpace(s.ServerURL), "/")
	return s, nil
}

func LoadDefault() (State, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return State{}, "", err
	}
	s, err := Load(path)
	if err != nil {
		return State{}, path, err
	}
	return s, path, nil
}

func Write(path string, s State) error {
	s.ServerURL = strings.TrimRight(strings.TrimSpace(s.ServerURL), "/")
	if s.ServerURL == "" {
		return errors.New("server_url is required")
	}
	if s.Version == 0 {
		s.Version = 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}

func WriteDefault(s State) (string, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", err
	}
	if err := Write(path, s); err != nil {
		return "", err
	}
	return path, nil
}
