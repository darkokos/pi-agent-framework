package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type config struct {
	RegistryDir string `json:"registryDir,omitempty"`
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

func readConfig() (*config, error) {
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

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func ResolveRoot() (string, error) {
	cfg, err := readConfig()
	if err != nil {
		return "", err
	}
	if cfg != nil && cfg.RegistryDir != "" {
		return cfg.RegistryDir, nil
	}

	return defaultRegistryDir()
}

func defaultRegistryDir() (string, error) {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(dir, "pi-agent-lab", "registry"), nil
}

func SetRegistryDir(path string) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config{RegistryDir: path}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, append(data, '\n'), 0o644)
}
