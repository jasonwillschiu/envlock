package recipients

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusActive  = "active"
	StatusRevoked = "revoked"
)

var (
	ErrDuplicateRecipient = errors.New("duplicate recipient")
	ErrRecipientNotFound  = errors.New("recipient not found")
)

type Recipient struct {
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
	Source      string    `json:"source,omitempty"`
	Note        string    `json:"note,omitempty"`
}

type Store struct {
	Version    int         `json:"version"`
	Recipients []Recipient `json:"recipients"`
}

func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Store{}, err
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return Store{}, err
	}
	if s.Version == 0 {
		s.Version = 1
	}
	return s, nil
}

func LoadOrInit(path string) (Store, error) {
	s, err := Load(path)
	if err == nil {
		return s, nil
	}
	if !os.IsNotExist(err) {
		return Store{}, err
	}
	return Store{Version: 1, Recipients: []Recipient{}}, nil
}

func Write(path string, s Store) error {
	if s.Version == 0 {
		s.Version = 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) ActiveCount() int {
	count := 0
	for _, r := range s.Recipients {
		if r.Status == StatusActive {
			count++
		}
	}
	return count
}

func (s *Store) Add(r Recipient) error {
	r.Name = strings.TrimSpace(r.Name)
	r.PublicKey = strings.TrimSpace(r.PublicKey)
	r.Fingerprint = strings.TrimSpace(r.Fingerprint)
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if r.Status == "" {
		r.Status = StatusActive
	}
	if r.Name == "" {
		return errors.New("recipient name is required")
	}
	if r.PublicKey == "" {
		return errors.New("recipient public key is required")
	}
	for _, existing := range s.Recipients {
		if strings.EqualFold(existing.Name, r.Name) {
			return fmt.Errorf("%w: name %q already exists", ErrDuplicateRecipient, r.Name)
		}
		if existing.PublicKey == r.PublicKey || existing.Fingerprint == r.Fingerprint {
			return fmt.Errorf("%w: recipient %q already exists", ErrDuplicateRecipient, r.Fingerprint)
		}
	}
	s.Recipients = append(s.Recipients, r)
	return nil
}

func (s *Store) Revoke(query string) (Recipient, error) {
	idx := s.findIndex(query)
	if idx < 0 {
		return Recipient{}, ErrRecipientNotFound
	}
	s.Recipients[idx].Status = StatusRevoked
	return s.Recipients[idx], nil
}

func (s *Store) Delete(query string) (Recipient, error) {
	idx := s.findIndex(query)
	if idx < 0 {
		return Recipient{}, ErrRecipientNotFound
	}
	removed := s.Recipients[idx]
	s.Recipients = append(s.Recipients[:idx], s.Recipients[idx+1:]...)
	return removed, nil
}

func (s *Store) findIndex(query string) int {
	q := strings.TrimSpace(query)
	for i, r := range s.Recipients {
		if strings.EqualFold(r.Name, q) || strings.EqualFold(r.Fingerprint, q) {
			return i
		}
	}
	return -1
}
