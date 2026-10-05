package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

var ErrAlreadyRegistered = errors.New("already registered")

type fileConfig struct {
	Registries []string `json:"registries,omitempty"`
}

func configPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir = filepath.Join(home, ".config")
	}

	return filepath.Join(dir, "pi-agent-lab", "config.json"), nil
}

func ListRegistered() ([]string, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return cfg.Registries, nil
}

func saveRegistries(paths []string) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(fileConfig{Registries: paths}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func Register(dir string) (*Manifest, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	m, err := ReadManifest(abs)
	if err != nil {
		return nil, err
	}

	paths, err := ListRegistered()
	if err != nil {
		return nil, err
	}

	if slices.Contains(paths, abs) {
		return m, ErrAlreadyRegistered
	}

	for _, p := range paths {
		other, err := ReadManifest(p)
		if err != nil || other.ID != m.ID {
			continue
		}

		return nil, fmt.Errorf("%s is the same registry as %s (id %s); register that one instead", abs, p, m.ID)
	}

	if err := saveRegistries(append(paths, abs)); err != nil {
		return nil, err
	}

	return m, nil
}

func InitAndRegister(dir, name string, now time.Time) (*Manifest, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if _, err := InitRegistry(abs, name, now); err != nil {
		return nil, err
	}

	m, err := Register(abs)
	if errors.Is(err, ErrAlreadyRegistered) {
		return m, nil
	}

	return m, err
}

func ResolveRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	if _, err := ReadManifest(cwd); err != nil {
		return "", err
	}

	return cwd, nil
}
