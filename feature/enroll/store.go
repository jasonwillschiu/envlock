package enroll

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	InviteStatusActive    = "active"
	InviteStatusUsed      = "used"
	InviteStatusRevoked   = "revoked"
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusRejected = "rejected"
)

var (
	ErrInvalidToken    = errors.New("invalid invite token")
	ErrInviteExpired   = errors.New("invite expired")
	ErrInviteNotFound  = errors.New("invite not found")
	ErrInviteUsed      = errors.New("invite already used")
	ErrRequestNotFound = errors.New("enrollment request not found")
)

type Invite struct {
	Version         int       `json:"version"`
	ID              string    `json:"id"`
	SecretHash      string    `json:"secret_hash"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedBy       string    `json:"created_by,omitempty"`
	UsedByRequestID string    `json:"used_by_request_id,omitempty"`
	UsedAt          time.Time `json:"used_at,omitempty"`
}

type Request struct {
	Version      int       `json:"version"`
	ID           string    `json:"id"`
	InviteID     string    `json:"invite_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	DecisionAt   time.Time `json:"decision_at,omitempty"`
	DecisionNote string    `json:"decision_note,omitempty"`

	DeviceName  string `json:"device_name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}

func InvitesDir(projectEnvlockDir string) string {
	return filepath.Join(projectEnvlockDir, "_enroll", "invites")
}

func RequestsDir(projectEnvlockDir string) string {
	return filepath.Join(projectEnvlockDir, "_enroll", "requests")
}

func InvitePath(projectEnvlockDir, id string) string {
	return filepath.Join(InvitesDir(projectEnvlockDir), id+".json")
}

func RequestPath(projectEnvlockDir, id string) string {
	return filepath.Join(RequestsDir(projectEnvlockDir), id+".json")
}

func CreateInvite(projectEnvlockDir string, ttl time.Duration, createdBy string) (Invite, string, string, error) {
	invite, token, err := NewInvite(ttl, createdBy)
	if err != nil {
		return Invite{}, "", "", err
	}
	path := InvitePath(projectEnvlockDir, invite.ID)
	if err := writeJSON(path, invite); err != nil {
		return Invite{}, "", "", err
	}
	return invite, token, path, nil
}

func NewInvite(ttl time.Duration, createdBy string) (Invite, string, error) {
	if ttl <= 0 {
		return Invite{}, "", errors.New("ttl must be greater than zero")
	}
	id, err := randomHex(8)
	if err != nil {
		return Invite{}, "", err
	}
	secret, err := randomTokenSecret(18)
	if err != nil {
		return Invite{}, "", err
	}
	now := time.Now().UTC()
	invite := Invite{
		Version:    1,
		ID:         id,
		SecretHash: secretHash(secret),
		Status:     InviteStatusActive,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		CreatedBy:  strings.TrimSpace(createdBy),
	}
	token := formatToken(invite.ID, secret)
	return invite, token, nil
}

func ParseToken(token string) (inviteID string, secret string, err error) {
	t := strings.TrimSpace(token)
	const prefix = "envlock-invite-"
	if !strings.HasPrefix(t, prefix) {
		return "", "", ErrInvalidToken
	}
	body := strings.TrimPrefix(t, prefix)
	parts := strings.Split(body, ".")
	if len(parts) != 2 {
		return "", "", ErrInvalidToken
	}
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", ErrInvalidToken
	}
	return parts[0], parts[1], nil
}

func LoadInvite(projectEnvlockDir, id string) (Invite, string, error) {
	path := InvitePath(projectEnvlockDir, id)
	invite, err := readInvite(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Invite{}, "", ErrInviteNotFound
		}
		return Invite{}, "", err
	}
	return invite, path, nil
}

func LoadInviteByToken(projectEnvlockDir, token string) (Invite, string, error) {
	id, secret, err := ParseToken(token)
	if err != nil {
		return Invite{}, "", err
	}
	invite, path, err := LoadInvite(projectEnvlockDir, id)
	if err != nil {
		return Invite{}, "", err
	}
	if invite.SecretHash != secretHash(secret) {
		return Invite{}, "", ErrInvalidToken
	}
	return invite, path, nil
}

func VerifyToken(invite Invite, token string) error {
	id, secret, err := ParseToken(token)
	if err != nil {
		return err
	}
	if strings.TrimSpace(invite.ID) != strings.TrimSpace(id) {
		return ErrInvalidToken
	}
	if invite.SecretHash != secretHash(secret) {
		return ErrInvalidToken
	}
	return nil
}

func CreateJoinRequest(projectEnvlockDir string, invite Invite, deviceName, publicKey, fingerprint string) (Request, string, error) {
	existing, err := ListRequests(projectEnvlockDir)
	if err != nil {
		return Request{}, "", err
	}
	req, err := NewJoinRequest(existing, invite, deviceName, publicKey, fingerprint)
	if err != nil {
		return Request{}, "", err
	}
	path := RequestPath(projectEnvlockDir, req.ID)
	if err := writeJSON(path, req); err != nil {
		return Request{}, "", err
	}
	return req, path, nil
}

func NewJoinRequest(existing []Request, invite Invite, deviceName, publicKey, fingerprint string) (Request, error) {
	if strings.TrimSpace(invite.ID) == "" {
		return Request{}, errors.New("invite id is required")
	}
	if isInviteExpired(invite, time.Now().UTC()) {
		return Request{}, ErrInviteExpired
	}
	if invite.Status == InviteStatusUsed {
		return Request{}, ErrInviteUsed
	}
	if invite.Status != "" && invite.Status != InviteStatusActive {
		return Request{}, fmt.Errorf("invite status is %s", invite.Status)
	}
	for _, r := range existing {
		if r.InviteID == invite.ID && r.Status == RequestStatusPending {
			return Request{}, fmt.Errorf("pending request already exists for invite %s", invite.ID)
		}
		if r.Fingerprint == strings.TrimSpace(fingerprint) && r.Status == RequestStatusPending {
			return Request{}, fmt.Errorf("device %s already has a pending request", r.DeviceName)
		}
	}

	id, err := randomHex(8)
	if err != nil {
		return Request{}, err
	}
	req := Request{
		Version:     1,
		ID:          id,
		InviteID:    invite.ID,
		Status:      RequestStatusPending,
		CreatedAt:   time.Now().UTC(),
		DeviceName:  strings.TrimSpace(deviceName),
		PublicKey:   strings.TrimSpace(publicKey),
		Fingerprint: strings.TrimSpace(fingerprint),
	}
	return req, nil
}

func ListRequests(projectEnvlockDir string) ([]Request, error) {
	dir := RequestsDir(projectEnvlockDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Request{}, nil
		}
		return nil, err
	}
	var out []Request
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		req, err := readRequest(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func LoadRequest(projectEnvlockDir, id string) (Request, string, error) {
	path := RequestPath(projectEnvlockDir, id)
	req, err := readRequest(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Request{}, "", ErrRequestNotFound
		}
		return Request{}, "", err
	}
	return req, path, nil
}

func WriteInvite(path string, invite Invite) error {
	if invite.Version == 0 {
		invite.Version = 1
	}
	return writeJSON(path, invite)
}

func WriteRequest(path string, req Request) error {
	if req.Version == 0 {
		req.Version = 1
	}
	return writeJSON(path, req)
}

func ValidateInviteForJoin(invite Invite, now time.Time) error {
	if invite.Status == InviteStatusUsed {
		return ErrInviteUsed
	}
	if invite.Status != "" && invite.Status != InviteStatusActive {
		return fmt.Errorf("invite status is %s", invite.Status)
	}
	if isInviteExpired(invite, now) {
		return ErrInviteExpired
	}
	return nil
}

func ValidateInviteForApproval(invite Invite) error {
	if invite.Status == InviteStatusUsed {
		return ErrInviteUsed
	}
	if invite.Status != "" && invite.Status != InviteStatusActive {
		return fmt.Errorf("invite status is %s", invite.Status)
	}
	return nil
}

func isInviteExpired(invite Invite, now time.Time) bool {
	return !invite.ExpiresAt.IsZero() && now.After(invite.ExpiresAt)
}

func formatToken(id, secret string) string {
	return "envlock-invite-" + id + "." + secret
}

func secretHash(secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomTokenSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func readInvite(path string) (Invite, error) {
	var v Invite
	if err := readJSON(path, &v); err != nil {
		return Invite{}, err
	}
	if v.Version == 0 {
		v.Version = 1
	}
	return v, nil
}

func readRequest(path string) (Request, error) {
	var v Request
	if err := readJSON(path, &v); err != nil {
		return Request{}, err
	}
	if v.Version == 0 {
		v.Version = 1
	}
	return v, nil
}

func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
