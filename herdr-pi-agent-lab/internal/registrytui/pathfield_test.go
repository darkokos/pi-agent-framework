package registrytui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func mkTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, d := range []string{"prompts", "playground", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"plan.md", "notes.txt", ".env"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestPathMatches_DirsFirstHiddenExcluded(t *testing.T) {
	root := mkTree(t)

	got := pathMatches(root + "/")
	want := []string{"playground/", "prompts/", "notes.txt", "plan.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	got = pathMatches(root + "/pl")
	want = []string{"playground/", "plan.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	if got = pathMatches(root + "/."); !slices.Equal(got, []string{".hidden/", ".env"}) {
		t.Fatalf("hidden entries should show once a dot is typed, got %v", got)
	}
}

func TestCompletePath_CompletesCommonPrefixThenStops(t *testing.T) {
	root := mkTree(t)

	f := newPathField("p", "", "", root+"/p")
	if f.completePath() || f.text() != root+"/p" {
		t.Fatalf("ambiguous prefix with nothing in common must not change the text, got %q", f.text())
	}

	f = newPathField("p", "", "", root+"/play")
	if !f.completePath() || f.text() != root+"/playground/" {
		t.Fatalf("expected descent into playground/, got %q", f.text())
	}

	f = newPathField("p", "", "", root+"/pr")
	if !f.completePath() || f.text() != root+"/prompts/" {
		t.Fatalf("expected descent into prompts/, got %q", f.text())
	}
}

func TestCompletePath_EmptyFieldDoesNotComplete(t *testing.T) {
	t.Setenv("HOME", mkTree(t))

	f := newPathField("p", "", "", "")
	if f.completePath() || f.text() != "" {
		t.Fatalf("tab on an empty field must move on, not insert a prefix; got %q", f.text())
	}
}

func TestResolvePath_OnlyExpandsTilde(t *testing.T) {
	t.Setenv("HOME", "/home/x")

	cases := map[string]string{
		"~":        "/home/x",
		"~/a/b":    "/home/x/a/b",
		"a/b":      "a/b",
		"/abs/one": "/abs/one",
	}
	for in, want := range cases {
		if got := resolvePath(in); got != want {
			t.Errorf("resolvePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPathMatches_OnlyForExplicitPaths(t *testing.T) {
	home := mkTree(t)
	t.Setenv("HOME", home)

	if got := pathMatches("~/"); !slices.Equal(got, []string{"playground/", "prompts/", "notes.txt", "plan.md"}) {
		t.Fatalf("~/ must list the home directory, got %v", got)
	}
	for _, in := range []string{"", "~", "pro", "./x", "pl"} {
		if got := pathMatches(in); got != nil {
			t.Errorf("pathMatches(%q) = %v, want no completions", in, got)
		}
	}
}

func TestCompletePath_TildeExpandsToTildeSlash(t *testing.T) {
	t.Setenv("HOME", mkTree(t))

	f := newPathField("p", "", "", "~")
	if !f.completePath() || f.text() != "~/" {
		t.Fatalf("got %q", f.text())
	}
}

func TestRequireFile(t *testing.T) {
	home := mkTree(t)
	t.Setenv("HOME", home)

	f := newPathField("Prompt File", "", "", "~/")
	if _, err := requireFile(f); err == nil {
		t.Fatal("a directory must be rejected")
	}

	f.setPathText("plan.md")
	if _, err := requireFile(f); err == nil {
		t.Fatal("a relative path must be rejected")
	}

	f.setPathText("~/nope.md")
	if _, err := requireFile(f); err == nil {
		t.Fatal("a missing file must be rejected")
	}

	f.setPathText("~/plan.md")
	if p, err := requireFile(f); err != nil || p != home+"/plan.md" {
		t.Fatalf("got %q, %v", p, err)
	}
}
