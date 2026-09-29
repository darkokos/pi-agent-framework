package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

func VersionSkillsDir(root, agentID, versionID string) string {
	return versionSkillsDir(root, agentID, versionID)
}

func copySkills(dst string, sources []string) ([]string, error) {
	var names []string

	for _, src := range sources {
		name := filepath.Base(filepath.Clean(src))
		if slices.Contains(names, name) {
			return nil, fmt.Errorf("two skills are both named %q", name)
		}

		if err := os.MkdirAll(dst, 0o755); err != nil {
			return nil, err
		}

		if err := copyTree(src, filepath.Join(dst, name), map[string]bool{}); err != nil {
			return nil, fmt.Errorf("copying skill %q: %w", src, err)
		}

		names = append(names, name)
	}

	return names, nil
}

func copyTree(src, dst string, ancestors map[string]bool) error {
	real, err := filepath.EvalSymlinks(src)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %w", src, err)
	}

	info, err := os.Stat(real)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return copyFile(real, dst, info.Mode().Perm())
	}

	if ancestors[real] {
		return fmt.Errorf("%s links back to a folder that contains it", src)
	}

	ancestors[real] = true
	defer delete(ancestors, real)

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(real)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}

		if err := copyTree(filepath.Join(real, e.Name()), filepath.Join(dst, e.Name()), ancestors); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()

		return err
	}

	return out.Close()
}
