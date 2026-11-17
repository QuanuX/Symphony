package enrollment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
)

func Enroll(layout paths.Layout, cfg config.Config) (bool, error) {
	if err := cfg.Validate(); err != nil {
		return false, err
	}
	if cfg.TOPSID != layout.TOPSID || cfg.Mode != string(layout.Scope) || cfg.Listen.Address != layout.Socket {
		return false, fmt.Errorf("configuration does not match Accordare producer layout")
	}
	data, err := config.Marshal(cfg)
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if err := ensureDirectory(layout.ConfigDir); err != nil {
		return false, err
	}
	if existing, err := os.ReadFile(layout.ConfigFile); err == nil {
		if bytes.Equal(existing, data) {
			return false, nil
		}
		return false, fmt.Errorf("Accordare producer enrollment differs; unenroll before replacement")
	} else if !os.IsNotExist(err) {
		return false, err
	}
	return true, writeAtomic(layout.ConfigFile, data)
}

func Unenroll(layout paths.Layout, purge bool) (bool, error) {
	// Resolve pending work before removing the configuration needed to recover
	// it. This applies even when mutable data is retained rather than purged.
	if err := RequireNoPending(layout); err != nil {
		return false, err
	}
	if active(layout.Socket) {
		return false, fmt.Errorf("Accordare producer is active; stop supervision before unenrollment")
	}
	changed := false
	if err := os.Remove(layout.ConfigFile); err == nil {
		changed = true
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if purge {
		for _, path := range []string{layout.IntentDir, layout.OutboxDir, layout.StateDir, layout.ConfigDir, layout.RuntimeDir} {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return false, err
			}
		}
	}
	return changed, nil
}

// RequireNoPending is read-only. Missing stores are empty, while unreadable,
// linked, or non-empty stores retain the package and its recovery configuration.
func RequireNoPending(layout paths.Layout) error {
	for _, pendingDir := range []string{layout.IntentDir, layout.OutboxDir} {
		if err := validatePendingAncestors(pendingDir); err != nil {
			return err
		}
		info, err := os.Lstat(pendingDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("Accordare pending state is unavailable or unsafe")
		}
		entries, err := os.ReadDir(pendingDir)
		if err != nil {
			return fmt.Errorf("inspect Accordare pending state: %w", err)
		}
		if len(entries) != 0 {
			return fmt.Errorf("refusing to remove Accordare enrollment or package with pending audit state; resolve it first")
		}
	}
	return nil
}

func validatePendingAncestors(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) == string(filepath.Separator) {
		return fmt.Errorf("Accordare pending path is invalid")
	}
	current := string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Clean(path), current), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("Accordare pending path contains an unavailable or linked directory")
		}
	}
	return nil
}

func active(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSocket != 0
}

func ensureDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("Accordare enrollment directory is unsafe")
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
