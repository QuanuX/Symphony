package lifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

type Result struct {
	Scope   stavpaths.Scope
	Binary  string
	Changed bool
}

// Install atomically installs only the executable. It deliberately creates no
// manifest, configuration, state root, socket, or schema-bearing artifact.
func Install(source string, scope stavpaths.Scope, force bool) (Result, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return Result{}, err
	}
	source, sourceDigest, err := sourceFile(source)
	if err != nil {
		return Result{}, err
	}
	if err := ensureDirectory(filepath.Dir(layout.Binary)); err != nil {
		return Result{}, err
	}

	existingDigest, exists, err := regularFileDigest(layout.Binary)
	if err != nil {
		return Result{}, fmt.Errorf("inspect installed binary: %w", err)
	}
	if exists && existingDigest == sourceDigest {
		return Result{Scope: scope, Binary: layout.Binary, Changed: false}, nil
	}
	if exists && !force {
		return Result{}, fmt.Errorf("installed binary differs; use --force to replace it")
	}
	if err := copyAtomic(source, layout.Binary, 0755); err != nil {
		return Result{}, err
	}
	return Result{Scope: scope, Binary: layout.Binary, Changed: true}, nil
}

// Uninstall removes only the executable. A differing installed digest fails
// closed unless the operator explicitly supplies --force.
func Uninstall(source string, scope stavpaths.Scope, force bool) (Result, error) {
	layout, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return Result{}, err
	}
	_, sourceDigest, sourceErr := sourceFile(source)
	existingDigest, exists, err := regularFileDigest(layout.Binary)
	if err != nil {
		return Result{}, fmt.Errorf("inspect installed binary: %w", err)
	}
	if !exists {
		return Result{Scope: scope, Binary: layout.Binary, Changed: false}, nil
	}
	if sourceErr != nil && !force {
		return Result{}, fmt.Errorf("verify invoking executable: %w", sourceErr)
	}
	if sourceErr == nil && existingDigest != sourceDigest && !force {
		return Result{}, fmt.Errorf("installed binary differs from invoking executable; use --force to remove it")
	}
	if err := os.Remove(layout.Binary); err != nil {
		return Result{}, fmt.Errorf("remove installed binary: %w", err)
	}
	if err := syncDirectory(filepath.Dir(layout.Binary)); err != nil {
		return Result{}, err
	}
	return Result{Scope: scope, Binary: layout.Binary, Changed: true}, nil
}

func sourceFile(path string) (string, string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", "", fmt.Errorf("resolve source executable: %w", err)
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", "", fmt.Errorf("inspect source executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("source executable is not a regular file")
	}
	digest, err := fileDigest(resolved)
	if err != nil {
		return "", "", err
	}
	return resolved, digest, nil
}

func regularFileDigest(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("path is not a regular file: %s", path)
	}
	digest, err := fileDigest(path)
	return digest, true, err
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ensureDirectory(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || path == string(filepath.Separator) {
		return fmt.Errorf("unsafe installation directory %q", path)
	}

	var missing []string
	current := path
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("installation path component is not a directory: %s", current)
			}
			break
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect installation directory %s: %w", current, err)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("no existing parent for installation directory %s", path)
		}
		current = parent
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0755); err != nil && !os.IsExist(err) {
			return fmt.Errorf("create installation directory %s: %w", missing[i], err)
		}
		info, err := os.Lstat(missing[i])
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("installation directory is unsafe: %s", missing[i])
		}
	}
	return nil
}

func copyAtomic(source, target string, mode os.FileMode) (err error) {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source executable: %w", err)
	}
	defer in.Close()

	temp, err := os.CreateTemp(filepath.Dir(target), ".symphony-stav-append-authority-*")
	if err != nil {
		return fmt.Errorf("create temporary executable: %w", err)
	}
	tempName := temp.Name()
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(tempName)
		}
	}()
	if err = temp.Chmod(mode); err != nil {
		return fmt.Errorf("set executable permissions: %w", err)
	}
	if _, err = io.Copy(temp, in); err != nil {
		return fmt.Errorf("copy executable: %w", err)
	}
	if err = temp.Sync(); err != nil {
		return fmt.Errorf("sync executable: %w", err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("close executable: %w", err)
	}
	if err = os.Rename(tempName, target); err != nil {
		return fmt.Errorf("activate executable: %w", err)
	}
	return syncDirectory(filepath.Dir(target))
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open installation directory for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync installation directory: %w", err)
	}
	return nil
}
