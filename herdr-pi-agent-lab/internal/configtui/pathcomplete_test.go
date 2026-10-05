package configtui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func withHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	for _, d := range []string{"projects", "project-old", "Documents"} {
		if err := os.Mkdir(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	return home
}

func TestTypedMatches_RelativePathsCompleteFromHome(t *testing.T) {
	withHome(t)

	got := typedMatches("pro")
	want := []string{"project-old/", "projects/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("typedMatches(\"pro\") = %v, want %v", got, want)
	}
}

func TestTypedMatches_EmptyListsHomeDirectories(t *testing.T) {
	withHome(t)

	got := typedMatches("")
	want := []string{"Documents/", "project-old/", "projects/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("typedMatches(\"\") = %v, want %v", got, want)
	}
}

func TestTypedMatches_ExplicitPathsStayExplicit(t *testing.T) {
	home := withHome(t)

	got := typedMatches(home + "/Doc")
	want := []string{home + "/Documents/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTypedMatches_FilesAreNotOffered(t *testing.T) {
	withHome(t)

	for _, m := range typedMatches("no") {
		if m == "notes.txt" {
			t.Fatal("a registry must be a directory, so files should not be suggested")
		}
	}
}

func TestExpandPath_RelativeIsRelativeToHome(t *testing.T) {
	home := withHome(t)

	got, err := expandPath("projects/reg")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "projects", "reg"); got != want {
		t.Fatalf("expandPath = %q, want %q", got, want)
	}
}
