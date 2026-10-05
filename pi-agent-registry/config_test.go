package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func isolateConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestInitRegistry_WritesAManifestAndAdoptsExistingData(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "agents", "old"), 0o755); err != nil {
		t.Fatal(err)
	}

	m, err := InitRegistry(dir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != filepath.Base(dir) {
		t.Fatalf("name defaults to the directory name, got %q", m.Name)
	}

	read, err := ReadManifest(dir)
	if err != nil || read.ID != m.ID {
		t.Fatalf("manifest did not round-trip: %+v, %v", read, err)
	}

	if _, err := os.Stat(filepath.Join(dir, "agents", "old")); err != nil {
		t.Fatalf("existing agents must be adopted, not removed: %v", err)
	}
}

func TestInitRegistry_RefusesAnExistingRegistry(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRegistry(dir, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := InitRegistry(dir, "", time.Now()); err == nil {
		t.Fatal("re-initialising must not replace an existing registry's identity")
	}
}

func TestResolveRoot_IsTheWorkingDirectoryWhenItHasAManifest(t *testing.T) {
	isolateConfig(t)

	dir := t.TempDir()
	if _, err := InitRegistry(dir, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := ResolveRoot()
	if err != nil || got != dir {
		t.Fatalf("got %q, %v; want %q", got, err, dir)
	}
}

func TestResolveRoot_ErrorsOutsideARegistry(t *testing.T) {
	isolateConfig(t)
	t.Chdir(t.TempDir())

	_, err := ResolveRoot()
	if _, ok := errors.AsType[*NotARegistryError](err); !ok {
		t.Fatalf("expected a NotARegistryError, got %v", err)
	}
}

func TestRegister_RecognisesAValidRegistryAndListsIt(t *testing.T) {
	isolateConfig(t)

	dir := t.TempDir()
	if _, err := InitRegistry(dir, "", time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := Register(dir); err != nil {
		t.Fatal(err)
	}

	paths, err := ListRegistered()
	if err != nil || len(paths) != 1 || paths[0] != dir {
		t.Fatalf("got %v, %v; want [%s]", paths, err, dir)
	}
}

func TestRegister_RefusesAPlainDirectory(t *testing.T) {
	isolateConfig(t)

	_, err := Register(t.TempDir())
	if _, ok := errors.AsType[*NotARegistryError](err); !ok {
		t.Fatalf("expected a NotARegistryError, got %v", err)
	}

	if paths, _ := ListRegistered(); len(paths) != 0 {
		t.Fatalf("a refused registry must not be listed, got %v", paths)
	}
}

func TestRegister_TwiceIsReportedNotDuplicated(t *testing.T) {
	isolateConfig(t)

	dir := t.TempDir()
	if _, err := InitRegistry(dir, "", time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := Register(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := Register(dir); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("expected ErrAlreadyRegistered, got %v", err)
	}

	if paths, _ := ListRegistered(); len(paths) != 1 {
		t.Fatalf("expected one entry, got %v", paths)
	}
}

func TestRegister_RefusesASecondCopyOfTheSameRegistry(t *testing.T) {
	isolateConfig(t)

	first := t.TempDir()
	if _, err := InitRegistry(first, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := Register(first); err != nil {
		t.Fatal(err)
	}

	second := t.TempDir()
	data, err := os.ReadFile(filepath.Join(first, ManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, ManifestFile), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Register(second); err == nil {
		t.Fatal("a copy with the same manifest ID must not be registered as a separate registry")
	}
}

func TestInitAndRegister_AdoptsADirectoryAlreadyListedWithoutAManifest(t *testing.T) {
	isolateConfig(t)

	dir := t.TempDir()
	if err := saveRegistries([]string{dir}); err != nil {
		t.Fatal(err)
	}

	if _, err := InitAndRegister(dir, "", time.Now()); err != nil {
		t.Fatalf("initialising an already-listed directory should succeed, got %v", err)
	}

	if paths, _ := ListRegistered(); len(paths) != 1 {
		t.Fatalf("expected the directory listed once, got %v", paths)
	}
}
