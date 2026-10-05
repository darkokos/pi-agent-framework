package tuistate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

const pluginID = "darkokos.pi-agent-lab"

var ErrNoActive = errors.New("no registry selected: choose one in Pi Agent Lab → Registries")

type state struct {
	Active string `json:"active,omitempty"`
}

var (
	dirOnce sync.Once
	dir     string
	dirErr  error
)

func configDir() (string, error) {
	dirOnce.Do(func() {
		out, err := exec.Command("herdr", "plugin", "config-dir", pluginID).Output()
		if err != nil {
			dirErr = fmt.Errorf("asking herdr for the plugin config directory: %w", err)

			return
		}

		dir = strings.TrimSpace(string(out))
	})

	return dir, dirErr
}

func path() (string, error) {
	d, err := configDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(d, "tui.json"), nil
}

func Load() (string, error) {
	p, err := path()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return "", fmt.Errorf("reading %s: %w", p, err)
	}

	return s.Active, nil
}

func Save(active string) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state{Active: active}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func ActiveRoot() (string, error) {
	active, err := Load()
	if err != nil {
		return "", err
	}
	if active == "" {
		return "", ErrNoActive
	}

	paths, err := registry.ListRegistered()
	if err != nil {
		return "", err
	}
	if !slices.Contains(paths, active) {
		return "", fmt.Errorf("%s is no longer registered: choose another in Pi Agent Lab → Registries", active)
	}

	if _, err := registry.ReadManifest(active); err != nil {
		return "", err
	}

	return active, nil
}
