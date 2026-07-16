package paths

import (
	"fmt"
	"os"
	"path/filepath"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

const BinaryName = "symphony-stav-append-authority"

// InstallLayout contains the single host-level artifact owned by the current
// lifecycle scaffold. No installation manifest is created because its schema
// has not been ratified.
type InstallLayout struct {
	Scope  Scope
	Binary string
}

// InstanceLayout resolves the owner-ratified per-TOPS namespace without
// creating it. Schema content, enrollment, listeners, and ledger storage remain
// separately gated.
type InstanceLayout struct {
	Scope      Scope
	TOPSID     string
	ConfigDir  string
	ConfigFile string
	StateDir   string
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
		return "", fmt.Errorf("unsupported scope %q: expected user or system", value)
	}
}

// ValidateTOPSID accepts only a canonical lowercase UUID. Display names are
// deliberately excluded from security paths and identifiers.
func ValidateTOPSID(value string) error {
	return stavprotocol.ValidateTOPSID(value)
}

func ResolveInstall(scope Scope) (InstallLayout, error) {
	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return InstallLayout{}, fmt.Errorf("resolve user home: %w", err)
		}
		return cleanInstall(InstallLayout{
			Scope:  scope,
			Binary: filepath.Join(home, ".local", "bin", BinaryName),
		})
	case ScopeSystem:
		return cleanInstall(InstallLayout{
			Scope:  scope,
			Binary: filepath.Join("/usr/local/bin", BinaryName),
		})
	default:
		return InstallLayout{}, fmt.Errorf("unsupported scope %q", scope)
	}
}

func ResolveInstance(scope Scope, topsID string) (InstanceLayout, error) {
	if err := ValidateTOPSID(topsID); err != nil {
		return InstanceLayout{}, err
	}

	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return InstanceLayout{}, fmt.Errorf("resolve user home: %w", err)
		}
		configBase := os.Getenv("XDG_CONFIG_HOME")
		if configBase == "" {
			configBase = filepath.Join(home, ".config")
		}
		stateBase := os.Getenv("XDG_STATE_HOME")
		if stateBase == "" {
			stateBase = filepath.Join(home, ".local", "state")
		}

		configDir := filepath.Join(configBase, "symphony", topsID, "stav")
		stateDir := filepath.Join(stateBase, "symphony", topsID, "stav")
		runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
		var runtimeDir string
		if runtimeBase == "" {
			runtimeDir = filepath.Join(stateDir, "run")
		} else {
			runtimeDir = filepath.Join(runtimeBase, "symphony", topsID, "stav")
		}
		return cleanInstance(InstanceLayout{
			Scope:      scope,
			TOPSID:     topsID,
			ConfigDir:  configDir,
			ConfigFile: filepath.Join(configDir, "append-authority.json"),
			StateDir:   stateDir,
			RuntimeDir: runtimeDir,
			Socket:     filepath.Join(runtimeDir, "append.sock"),
		})
	case ScopeSystem:
		configDir := filepath.Join("/etc/symphony", topsID, "stav")
		stateDir := filepath.Join("/var/lib/symphony", topsID, "stav")
		runtimeDir := filepath.Join("/run/symphony", topsID, "stav")
		return cleanInstance(InstanceLayout{
			Scope:      scope,
			TOPSID:     topsID,
			ConfigDir:  configDir,
			ConfigFile: filepath.Join(configDir, "append-authority.json"),
			StateDir:   stateDir,
			RuntimeDir: runtimeDir,
			Socket:     filepath.Join(runtimeDir, "append.sock"),
		})
	default:
		return InstanceLayout{}, fmt.Errorf("unsupported scope %q", scope)
	}
}

func cleanInstall(layout InstallLayout) (InstallLayout, error) {
	if err := cleanAbsolute(&layout.Binary); err != nil {
		return InstallLayout{}, err
	}
	return layout, nil
}

func cleanInstance(layout InstanceLayout) (InstanceLayout, error) {
	values := []*string{&layout.ConfigDir, &layout.ConfigFile, &layout.StateDir, &layout.RuntimeDir, &layout.Socket}
	for _, value := range values {
		if err := cleanAbsolute(value); err != nil {
			return InstanceLayout{}, err
		}
	}
	return layout, nil
}

func cleanAbsolute(value *string) error {
	*value = filepath.Clean(*value)
	if !filepath.IsAbs(*value) || *value == string(filepath.Separator) {
		return fmt.Errorf("unsafe non-absolute layout path %q", *value)
	}
	return nil
}
