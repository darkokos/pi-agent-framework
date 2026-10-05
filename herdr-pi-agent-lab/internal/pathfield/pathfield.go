package pathfield

import (
	"os"
	"path/filepath"
	"strings"
)

func SplitPath(text string) (dir, base string) {
	i := strings.LastIndex(text, "/")

	return text[:i+1], text[i+1:]
}

func IsExplicit(p string) bool {
	return p == "~" || strings.HasPrefix(p, "~/") || filepath.IsAbs(p)
}

func Resolve(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}

	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	default:
		return p
	}
}

func Matches(text string, dirsOnly bool) []string {
	if !IsExplicit(text) || text == "~" {
		return nil
	}

	dir, base := SplitPath(text)

	listDir := Resolve(dir)

	entries, err := os.ReadDir(listDir)
	if err != nil {
		return nil
	}

	var dirs, files []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
			continue
		}

		isDir := e.IsDir()
		if e.Type()&os.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(listDir, name)); err == nil && st.IsDir() {
				isDir = true
			}
		}

		switch {
		case isDir:
			dirs = append(dirs, dir+name+"/")
		case !dirsOnly:
			files = append(files, dir+name)
		}
	}

	return append(dirs, files...)
}

func LongestCommonPrefix(names []string) string {
	if len(names) == 0 {
		return ""
	}

	prefix := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}

	return prefix
}
