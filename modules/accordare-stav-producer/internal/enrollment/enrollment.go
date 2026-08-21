package enrollment

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

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
		for _, pendingDir := range []string{layout.IntentDir, layout.OutboxDir} {
			entries, err := os.ReadDir(pendingDir)
			if err == nil && len(entries) != 0 {
				return false, fmt.Errorf("refusing to purge non-empty Accordare audit state; resolve it first")
			}
		}
		for _, path := range []string{layout.IntentDir, layout.OutboxDir, layout.StateDir, layout.ConfigDir, layout.RuntimeDir} {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return false, err
			}
		}
	}
	return changed, nil
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
