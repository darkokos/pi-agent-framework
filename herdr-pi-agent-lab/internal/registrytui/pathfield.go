package registrytui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/pathfield"
)

func splitPath(text string) (dir, base string) { return pathfield.SplitPath(text) }

func isExplicitPath(p string) bool { return pathfield.IsExplicit(p) }

func resolvePath(p string) string { return pathfield.Resolve(p) }

func pathMatches(text string) []string {
	dir, _ := splitPath(text)

	matches := pathfield.Matches(text, false)
	if matches == nil {
		return nil
	}

	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, strings.TrimPrefix(m, dir))
	}

	return names
}

func pathBrowseMatches(relBase, query string) []string {
	switch query {
	case "~":
		return []string{"~/"}
	case ".":
		return []string{"./", "../"}
	case "..":
		return []string{"../"}
	}

	dir, base := splitPath(query)

	var listDir string
	switch {
	case strings.HasPrefix(dir, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return []string{}
		}

		listDir = filepath.Join(home, dir[2:])
	case strings.HasPrefix(dir, "/"):
		listDir = dir
	case strings.HasPrefix(dir, "./") || strings.HasPrefix(dir, "../"):
		if relBase == "" {
			return []string{}
		}

		listDir = filepath.Join(relBase, dir)
	default:
		return nil
	}

	entries, err := os.ReadDir(listDir)
	if err != nil {
		return []string{}
	}

	var dirs, files []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, base) || (strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".")) {
			continue
		}

		isDir := e.IsDir()
		if e.Type()&os.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(listDir, name)); err == nil && st.IsDir() {
				isDir = true
			}
		}

		if isDir {
			dirs = append(dirs, dir+name+"/")
		} else {
			files = append(files, dir+name)
		}
	}

	return append(dirs, files...)
}

func longestCommonPrefix(names []string) string { return pathfield.LongestCommonPrefix(names) }

func pathKind(text string) string {
	if !isExplicitPath(text) {
		return ""
	}

	info, err := os.Stat(resolvePath(text))
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return "dir"
	}

	return "file"
}

func newPathField(label, help, placeholder, value string) *formField {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetValue(value)

	f := &formField{label: label, help: help, kind: kindPath, input: ti}
	f.refilterPath()

	return f
}

func (f *formField) refilterPath() {
	f.pickerMatches = pathMatches(f.text())
	f.pickerCursor = 0
}

func (f *formField) setPathText(s string) {
	f.input.SetValue(s)
	f.input.CursorEnd()
	f.refilterPath()
}

func (f *formField) applyPathChoice(name string) {
	dir, _ := splitPath(f.text())
	f.setPathText(dir + name)
}

func (f *formField) completePath() bool {
	text := f.text()
	if text == "" {
		return false
	}

	if text == "~" {
		f.setPathText("~/")

		return true
	}

	dir, base := splitPath(text)

	lcp := longestCommonPrefix(f.pickerMatches)
	if len(lcp) <= len(base) {
		return false
	}

	f.setPathText(dir + lcp)

	return true
}

func (f *formField) pathValue() string {
	if f.text() == "" {
		return ""
	}

	return resolvePath(f.text())
}

func requireFile(f *formField) (string, error) {
	path := f.pathValue()
	if path != "" && !isExplicitPath(f.text()) {
		return "", fmt.Errorf("%s: path must start with ~/ or /", f.label)
	}

	info, err := os.Stat(path)
	switch {
	case path == "" || os.IsNotExist(err):
		return "", fmt.Errorf("%s: no such file", f.label)
	case err != nil:
		return "", fmt.Errorf("%s: %w", f.label, err)
	case info.IsDir():
		return "", fmt.Errorf("%s: %s is a directory, pick a file", f.label, path)
	}

	return path, nil
}
