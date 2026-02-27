package keys

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

type GeneratedIdentity struct {
	Identity   *age.X25519Identity
	Recipient  *age.X25519Recipient
	DeviceName string
}

type Metadata struct {
	DeviceName string
}

func Generate(deviceName string) (GeneratedIdentity, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return GeneratedIdentity{}, err
	}
	return GeneratedIdentity{
		Identity:   id,
		Recipient:  id.Recipient(),
		DeviceName: strings.TrimSpace(deviceName),
	}, nil
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "envlock"), nil
}

func keysDir() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "keys"), nil
}

func DefaultKeyPath(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = "default"
	}
	if strings.Contains(name, "/") || strings.Contains(name, string(os.PathSeparator)) {
		return "", errors.New("key name must not contain path separators")
	}
	dir, err := keysDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".agekey"), nil
}

func WriteIdentity(path string, generated GeneratedIdentity, force bool) error {
	if generated.Identity == nil {
		return errors.New("missing identity")
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("key already exists at %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	content := generated.Identity.String()
	if generated.DeviceName != "" {
		content = fmt.Sprintf("# envlock-device: %s\n%s", generated.DeviceName, content)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return err
	}
	return nil
}

func LoadIdentity(path string) (*age.X25519Identity, Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Metadata{}, err
	}
	var meta Metadata
	var lines []string
	s := bufio.NewScanner(strings.NewReader(string(data)))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "# envlock-device:") {
			meta.DeviceName = strings.TrimSpace(strings.TrimPrefix(line, "# envlock-device:"))
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := s.Err(); err != nil {
		return nil, Metadata{}, err
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			id, err := age.ParseX25519Identity(line)
			if err != nil {
				return nil, Metadata{}, err
			}
			return id, meta, nil
		}
	}
	return nil, Metadata{}, errors.New("no AGE-SECRET-KEY found")
}

func ValidateRecipientString(pub string) error {
	_, err := age.ParseX25519Recipient(strings.TrimSpace(pub))
	return err
}

func Fingerprint(pub string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(pub)))
	return hex.EncodeToString(sum[:8])
}
