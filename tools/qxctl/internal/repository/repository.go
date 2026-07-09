package repository

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot locates the Symphony repository root by looking for README.md and INTENT.md.
func FindRoot(start string) (string, error) {
	current := start
	for {
		hasReadme := IsFile(filepath.Join(current, "README.md"))
		hasIntent := IsFile(filepath.Join(current, "INTENT.md"))
		hasModules := IsDir(filepath.Join(current, "modules"))

		if hasReadme && hasIntent && hasModules {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("Symphony repository root not found in any parent directory")
}

// IsFile checks if the given path is a regular file.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// IsDir checks if the given path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
