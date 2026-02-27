package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrPendingLoginNotFound = errors.New("pending login not found")
	ErrPendingLoginExpired  = errors.New("pending login expired")
	ErrInvalidCode          = errors.New("invalid login code")
	ErrCodeExpired          = errors.New("login code expired")
	ErrTokenNotFound        = errors.New("access token not found")
	ErrTokenExpired         = errors.New("access token expired")
)

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type PendingCLILogin struct {
	State       string
	CallbackURL string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type CLILoginCode struct {
	Code      string
	State     string
	User      User
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type AccessToken struct {
	Token        string
	User         User
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RefreshToken string
}

type MemoryStore struct {
	mu            sync.Mutex
	pending       map[string]PendingCLILogin
	codes         map[string]CLILoginCode
	accessTokens  map[string]AccessToken
	refreshTokens map[string]AccessToken
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		pending:       map[string]PendingCLILogin{},
		codes:         map[string]CLILoginCode{},
		accessTokens:  map[string]AccessToken{},
		refreshTokens: map[string]AccessToken{},
	}
}

func (s *MemoryStore) StartCLILogin(callbackURL string, ttl time.Duration) (PendingCLILogin, error) {
	if ttl <= 0 {
		return PendingCLILogin{}, fmt.Errorf("ttl must be > 0")
	}
	state, err := randomHex(16)
	if err != nil {
		return PendingCLILogin{}, err
	}
	now := time.Now().UTC()
	item := PendingCLILogin{
		State:       state,
		CallbackURL: callbackURL,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	s.pending[state] = item
	return item, nil
}

func (s *MemoryStore) GetPendingCLILogin(state string, now time.Time) (PendingCLILogin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	item, ok := s.pending[state]
	if !ok {
		return PendingCLILogin{}, ErrPendingLoginNotFound
	}
	if now.After(item.ExpiresAt) {
		delete(s.pending, state)
		return PendingCLILogin{}, ErrPendingLoginExpired
	}
	return item, nil
}

func (s *MemoryStore) IssueCodeForState(state string, user User, ttl time.Duration, now time.Time) (CLILoginCode, error) {
	if ttl <= 0 {
		return CLILoginCode{}, fmt.Errorf("ttl must be > 0")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	p, ok := s.pending[state]
	if !ok {
		return CLILoginCode{}, ErrPendingLoginNotFound
	}
	if now.After(p.ExpiresAt) {
		delete(s.pending, state)
		return CLILoginCode{}, ErrPendingLoginExpired
	}
	codeRaw, err := randomHex(8)
	if err != nil {
		return CLILoginCode{}, err
	}
	code := "envlock-code-" + codeRaw
	item := CLILoginCode{
		Code:      code,
		State:     state,
		User:      user,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	s.codes[code] = item
	return item, nil
}

func (s *MemoryStore) ExchangeCode(code, state string, accessTTL, refreshTTL time.Duration, now time.Time) (AccessToken, error) {
	if accessTTL <= 0 || refreshTTL <= 0 {
		return AccessToken{}, fmt.Errorf("token ttl must be > 0")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)

	item, ok := s.codes[code]
	if !ok {
		return AccessToken{}, ErrInvalidCode
	}
	if state != "" && item.State != "" && item.State != state {
		return AccessToken{}, ErrInvalidCode
	}
	if now.After(item.ExpiresAt) {
		delete(s.codes, code)
		return AccessToken{}, ErrCodeExpired
	}
	if item.UsedAt != nil {
		return AccessToken{}, ErrInvalidCode
	}

	accessToken, err := randomToken("atk_")
	if err != nil {
		return AccessToken{}, err
	}
	refreshToken, err := randomToken("rtk_")
	if err != nil {
		return AccessToken{}, err
	}
	out := AccessToken{
		Token:        accessToken,
		User:         item.User,
		CreatedAt:    now,
		ExpiresAt:    now.Add(accessTTL),
		RefreshToken: refreshToken,
	}
	s.accessTokens[accessToken] = out
	s.refreshTokens[refreshToken] = AccessToken{
		Token:     refreshToken,
		User:      item.User,
		CreatedAt: now,
		ExpiresAt: now.Add(refreshTTL),
	}
	usedAt := now
	item.UsedAt = &usedAt
	s.codes[code] = item
	delete(s.pending, item.State)
	return out, nil
}

func (s *MemoryStore) ValidateAccessToken(token string, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	t, ok := s.accessTokens[token]
	if !ok {
		return User{}, ErrTokenNotFound
	}
	if now.After(t.ExpiresAt) {
		delete(s.accessTokens, token)
		return User{}, ErrTokenExpired
	}
	return t.User, nil
}

func (s *MemoryStore) cleanupLocked(now time.Time) {
	for k, v := range s.pending {
		if now.After(v.ExpiresAt) {
			delete(s.pending, k)
		}
	}
	for k, v := range s.codes {
		if now.After(v.ExpiresAt.Add(5 * time.Minute)) {
			delete(s.codes, k)
		}
	}
	for k, v := range s.accessTokens {
		if now.After(v.ExpiresAt) {
			delete(s.accessTokens, k)
		}
	}
	for k, v := range s.refreshTokens {
		if now.After(v.ExpiresAt) {
			delete(s.refreshTokens, k)
		}
	}
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomToken(prefix string) (string, error) {
	s, err := randomHex(24)
	if err != nil {
		return "", err
	}
	return prefix + s, nil
}
