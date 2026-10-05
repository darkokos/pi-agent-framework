package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
)

const ManifestFile = "pi-agent-lab.toml"

const manifestFormatVersion = 1

type Manifest struct {
	FormatVersion int    `toml:"format_version"`
	ID            string `toml:"id"`
	Name          string `toml:"name"`
	CreatedAt     string `toml:"created_at"`
}

type NotARegistryError struct {
	Dir string
	Err error
}

func (e *NotARegistryError) Error() string {
	return fmt.Sprintf("%s is not a registry (no valid %s): run `pi-agent-lab registry init` there to create one", e.Dir, ManifestFile)
}

func (e *NotARegistryError) Unwrap() error { return e.Err }

func ReadManifest(dir string) (*Manifest, error) {
	var m Manifest
	if _, err := toml.DecodeFile(filepath.Join(dir, ManifestFile), &m); err != nil {
		return nil, &NotARegistryError{Dir: dir, Err: err}
	}

	if m.FormatVersion != manifestFormatVersion {
		return nil, &NotARegistryError{Dir: dir, Err: fmt.Errorf("unsupported format_version %d (this build reads %d)", m.FormatVersion, manifestFormatVersion)}
	}
	if m.ID == "" {
		return nil, &NotARegistryError{Dir: dir, Err: fmt.Errorf("manifest has no id")}
	}

	return &m, nil
}

func InitRegistry(dir, name string, now time.Time) (*Manifest, error) {
	path := filepath.Join(dir, ManifestFile)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("%s is already a registry", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	if name == "" {
		name = filepath.Base(dir)
	}

	m := &Manifest{
		FormatVersion: manifestFormatVersion,
		ID:            uuid.NewString(),
		Name:          name,
		CreatedAt:     now.UTC().Format(time.RFC3339),
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := toml.NewEncoder(f).Encode(m); err != nil {
		_ = f.Close()

		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	return m, nil
}
