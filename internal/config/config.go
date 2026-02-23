package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

var ErrProjectNotFound = errors.New("envlock project config not found")

type Project struct {
	Version  int    `toml:"version"`
	AppName  string `toml:"app_name"`
	Bucket   string `toml:"bucket"`
	Prefix   string `toml:"prefix"`
	Endpoint string `toml:"endpoint,omitempty"`
}

func DefaultPrefix(appName string) string {
	clean := strings.Trim(strings.TrimSpace(appName), "/")
	return fmt.Sprintf("envlock/%s", clean)
}

func ProjectDirPath(base string) string {
	return filepath.Join(base, ".envlock")
}

func ProjectFilePath(base string) string {
	return filepath.Join(ProjectDirPath(base), "project.toml")
}

func WriteProject(path string, p Project) error {
	if p.Version == 0 {
		p.Version = 1
	}
	if strings.TrimSpace(p.AppName) == "" {
		return errors.New("project app_name is required")
	}
	if strings.TrimSpace(p.Bucket) == "" {
		return errors.New("project bucket is required")
	}
	if strings.TrimSpace(p.Prefix) == "" {
		p.Prefix = DefaultPrefix(p.AppName)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(p)
}

func LoadProject(path string) (Project, error) {
	var p Project
	if _, err := toml.DecodeFile(path, &p); err != nil {
		return Project{}, err
	}
	if p.Version == 0 {
		p.Version = 1
	}
	return p, nil
}

func LoadProjectFromCWD() (Project, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Project{}, "", err
	}
	projPath := ProjectFilePath(cwd)
	if _, err := os.Stat(projPath); err != nil {
		if os.IsNotExist(err) {
			return Project{}, "", ErrProjectNotFound
		}
		return Project{}, "", err
	}
	p, err := LoadProject(projPath)
	if err != nil {
		return Project{}, "", err
	}
	return p, projPath, nil
}
