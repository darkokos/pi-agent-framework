package validate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledExtensionSources_PinsWhatIsInstalled(t *testing.T) {
	dir := t.TempDir()

	npmDir := filepath.Join(dir, "node_modules", "@tmustier", "usage")
	if err := os.MkdirAll(npmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npmDir, "package.json"), []byte(`{"name":"@tmustier/usage","version":"2.4.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(dir, "git", "github.com", "user", "repo")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "x"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", gitDir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	head, err := exec.Command("git", "-C", gitDir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(head))

	listing := fmt.Sprintf(
		"\x1b[1mUser packages:\x1b[22m\n"+
			"  npm:@tmustier/usage\n\x1b[2m    %s\x1b[22m\n"+
			"  npm:@pinned/already@1.0.0\n\x1b[2m    /nowhere\x1b[22m\n"+
			"  git:github.com/user/repo\n\x1b[2m    %s\x1b[22m\n"+
			"  /local/dev/extension\n\x1b[2m    /local/dev/extension\x1b[22m\n",
		npmDir, gitDir,
	)

	bin := t.TempDir()
	script := "#!/bin/sh\ncat <<'EOF'\n" + listing + "EOF\n"
	if err := os.WriteFile(filepath.Join(bin, "pi"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	got := InstalledExtensionSources()
	want := []string{
		"npm:@tmustier/usage@2.4.1",
		"npm:@pinned/already@1.0.0",
		"git:github.com/user/repo@" + commit,
	}

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
