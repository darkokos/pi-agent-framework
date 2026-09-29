package registrytui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRegistryPathMatches(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "registry")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "my-ext"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "other-ext.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := registryPathMatches(root, "npm:foo"); got != nil {
		t.Fatalf("a non-path query must return nil (not a path query), got %v", got)
	}

	if got := registryPathMatches(root, "."); !slices.Equal(got, []string{"./", "../"}) {
		t.Fatalf("got %v", got)
	}
	if got := registryPathMatches(root, ".."); !slices.Equal(got, []string{"../"}) {
		t.Fatalf("got %v", got)
	}

	got := registryPathMatches(root, "../")
	if !slices.Contains(got, "../my-ext/") || !slices.Contains(got, "../other-ext.ts") {
		t.Fatalf("got %v", got)
	}

	if got := registryPathMatches(root, "../my"); !slices.Equal(got, []string{"../my-ext/"}) {
		t.Fatalf("got %v", got)
	}

	if got := registryPathMatches(root, "../nope"); len(got) != 0 {
		t.Fatalf("a query with no matches must be an empty, non-nil slice, got %v", got)
	}
}

func TestExtensionSourcesField_BrowsesRegistryRelativePaths(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "registry")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "my-ext"), 0o755); err != nil {
		t.Fatal(err)
	}

	fields := versionSettingsFields(root)
	extField := fields[6]
	if extField.label != "Extension Sources" {
		t.Fatalf("field 6 is %q, not Extension Sources", extField.label)
	}

	extField.input.SetValue("npm:@foo/bar@1.2.3,../")
	extField.refilterPicker()
	if !slices.Contains(extField.pickerMatches, "../my-ext/") {
		t.Fatalf("typing ../ after a comma must browse the registry's parent, got %v", extField.pickerMatches)
	}

	extField.input.SetValue("npm:")
	extField.refilterPicker()
	if slices.Contains(extField.pickerMatches, "../my-ext/") {
		t.Fatalf("a non-path query must fall back to the installed-package suggestions, got %v", extField.pickerMatches)
	}
}
