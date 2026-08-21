package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

type Layout struct {
	Scope      Scope
	TOPSID     string
	ConfigDir  string
	ConfigFile string
	StateDir   string
	OutboxDir  string
	RuntimeDir string
	Socket     string
}

func ParseScope(value string) (Scope, error) {
	switch Scope(value) {
	case ScopeUser:
		return ScopeUser, nil
	case ScopeSystem:
		return ScopeSystem, nil
	default:
		return "", fmt.Errorf("unsupported scope %q", value)
	}
}

func Resolve(scope Scope, topsID string) (Layout, error) {
	if err := stavprotocol.ValidateTOPSID(topsID); err != nil {
		return Layout{}, err
	}
	var configBase, stateBase, runtimeBase string
	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return Layout{}, err
		}
		configBase = os.Getenv("XDG_CONFIG_HOME")
		if configBase == "" {
			configBase = filepath.Join(home, ".config")
		}
		stateBase = os.Getenv("XDG_STATE_HOME")
		if stateBase == "" {
			stateBase = filepath.Join(home, ".local", "state")
		}
		runtimeBase = os.Getenv("XDG_RUNTIME_DIR")
		if runtimeBase == "" {
			runtimeBase = stateBase
		}
	case ScopeSystem:
		configBase = "/etc"
		stateBase = "/var/lib"
		if runtime.GOOS == "darwin" {
			runtimeBase = "/var/run"
		} else {
			runtimeBase = "/run"
		}
	default:
		return Layout{}, fmt.Errorf("unsupported scope %q", scope)
	}
	configDir := filepath.Join(configBase, "symphony", topsID, "accordare-stav-producer")
	stateDir := filepath.Join(stateBase, "symphony", topsID, "accordare-stav-producer")
	runtimeDir := filepath.Join(runtimeBase, "symphony", topsID, "accordare-stav-producer")
	values := []string{configDir, stateDir, runtimeDir}
	for _, value := range values {
		if !filepath.IsAbs(value) || filepath.Clean(value) == string(filepath.Separator) {
			return Layout{}, fmt.Errorf("unsafe Accordare producer path %q", value)
		}
	}
	return Layout{
		Scope: scope, TOPSID: topsID, ConfigDir: filepath.Clean(configDir),
		ConfigFile: filepath.Join(configDir, "config.json"), StateDir: filepath.Clean(stateDir),
		OutboxDir: filepath.Join(stateDir, "outbox-v1"), RuntimeDir: filepath.Clean(runtimeDir),
		Socket: filepath.Join(runtimeDir, "submit.sock"),
	}, nil
}
