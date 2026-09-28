package registry

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveRoot_UnsetIsAnErrorCarryingTheSuggestion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "/data")

	root, err := ResolveRoot()
	if root != "" {
		t.Fatalf("an unset registry must not resolve to any directory, got %q", root)
	}

	var noReg *NoRegistryError
	if !errors.As(err, &noReg) {
		t.Fatalf("expected a NoRegistryError, got %v", err)
	}

	want := filepath.Join("/data", "pi-agent-lab", "registry")
	if noReg.Suggested != want {
		t.Fatalf("suggested = %q, want %q", noReg.Suggested, want)
	}
}

func TestResolveRoot_UsesTheExplicitlySetDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()
	if err := SetRegistryDir(dir); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveRoot()
	if err != nil || got != dir {
		t.Fatalf("got %q, %v; want %q", got, err, dir)
	}
}
