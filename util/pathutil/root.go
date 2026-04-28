package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const dirPerm os.FileMode = 0o750

// RelativeUnder returns the name of target relative to rootDir after proving it
// does not escape rootDir.
func RelativeUnder(rootDir, target string) (string, error) {
	if strings.TrimSpace(rootDir) == "" {
		return "", fmt.Errorf("empty root directory")
	}
	if strings.TrimSpace(target) == "" {
		return "", fmt.Errorf("empty target path")
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}

	absTarget := target
	if !filepath.IsAbs(absTarget) {
		absTarget = filepath.Join(absRoot, absTarget)
	}
	absTarget, err = filepath.Abs(absTarget)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return rel, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %q escapes root %q", target, rootDir)
	}
	return rel, nil
}

func OpenUnder(rootDir, target string) (*os.File, error) {
	rel, err := RelativeUnder(rootDir, target)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Open(rel)
}

func OpenFileUnder(rootDir, target string, flag int, perm os.FileMode) (*os.File, error) {
	rel, err := RelativeUnder(rootDir, target)
	if err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		if err := os.MkdirAll(rootDir, dirPerm); err != nil {
			return nil, err
		}
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	if flag&os.O_CREATE != 0 {
		dir := filepath.Dir(rel)
		if dir != "." {
			if err := root.MkdirAll(dir, dirPerm); err != nil {
				return nil, err
			}
		}
	}
	return root.OpenFile(rel, flag, perm)
}

func ReadFileUnder(rootDir, target string) ([]byte, error) {
	rel, err := RelativeUnder(rootDir, target)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(rel)
}
