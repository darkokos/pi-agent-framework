package validate

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type ExtensionSource struct {
	Kind     string
	Pinned   bool
	Portable bool
}

var (
	exactVersion = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	ansiEscape   = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func ParseExtensionSource(s string) (ExtensionSource, error) {
	s = strings.TrimSpace(s)

	switch {
	case s == "":
		return ExtensionSource{}, errors.New("extension source is empty")
	case strings.HasPrefix(s, "npm:"):
		return parseNpmSource(s)
	case isGitSource(s):
		return parseGitSource(s)
	case s == "~" || strings.HasPrefix(s, "~/") || filepath.IsAbs(s):
		return ExtensionSource{Kind: "path", Pinned: true, Portable: false}, nil
	default:
		return ExtensionSource{Kind: "path", Pinned: true, Portable: true}, nil
	}
}

func splitNpmSpec(spec string) (name, version string) {
	from := 0
	if strings.HasPrefix(spec, "@") {
		from = 1
	}

	if i := strings.Index(spec[from:], "@"); i >= 0 {
		return spec[:from+i], spec[from+i+1:]
	}

	return spec, ""
}

func parseNpmSource(s string) (ExtensionSource, error) {
	name, version := splitNpmSpec(strings.TrimPrefix(s, "npm:"))
	if name == "" || name == "@" {
		return ExtensionSource{}, fmt.Errorf("%q is missing a package name", s)
	}

	return ExtensionSource{Kind: "npm", Pinned: exactVersion.MatchString(version), Portable: true}, nil
}

func isGitSource(s string) bool {
	for _, prefix := range []string{"git:", "https://", "http://", "ssh://", "git@"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}

	return false
}

func parseGitSource(s string) (ExtensionSource, error) {
	last := s[strings.LastIndex(s, "/")+1:]
	if last == "" || strings.HasSuffix(s, ":") {
		return ExtensionSource{}, fmt.Errorf("%q does not name a repository", s)
	}

	at := strings.Index(last, "@")

	return ExtensionSource{Kind: "git", Pinned: at >= 0 && at < len(last)-1, Portable: true}, nil
}

func InstalledExtensionSources() []string {
	out, err := exec.Command("pi", "list").Output()
	if err != nil {
		return nil
	}

	var (
		sources []string
		pending string
	)

	scanner := bufio.NewScanner(strings.NewReader(ansiEscape.ReplaceAllString(string(out), "")))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "    "):
			if pending != "" {
				if pinned := pinnedForm(pending, strings.TrimSpace(line)); pinned != "" {
					sources = append(sources, pinned)
				}
			}

			pending = ""
		case strings.HasPrefix(line, "  "):
			pending = strings.TrimSpace(line)
		default:
			pending = ""
		}
	}

	return sources
}

func pinnedForm(source, installedAt string) string {
	parsed, err := ParseExtensionSource(source)
	if err != nil || parsed.Kind == "path" {
		return ""
	}
	if parsed.Pinned {
		return source
	}

	switch parsed.Kind {
	case "npm":
		data, err := os.ReadFile(filepath.Join(installedAt, "package.json"))
		if err != nil {
			return ""
		}

		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &pkg) != nil || !exactVersion.MatchString(pkg.Version) {
			return ""
		}

		name, _ := splitNpmSpec(strings.TrimPrefix(source, "npm:"))

		return "npm:" + name + "@" + pkg.Version
	default:
		commit, err := exec.Command("git", "-C", installedAt, "rev-parse", "HEAD").Output()
		if err != nil {
			return ""
		}

		return source + "@" + strings.TrimSpace(string(commit))
	}
}
