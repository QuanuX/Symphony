package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

// InstallLayout contains host-level files shared by every TOPS enrollment.
type InstallLayout struct {
	Scope           Scope
	Binary          string
	StateDir        string
	InstallManifest string
}

// InstanceLayout contains files isolated to one immutable TOPS identity.
type InstanceLayout struct {
	Scope              Scope
	TOPSID             string
	ConfigDir          string
	ConfigFile         string
	StateDir           string
	RuntimeDir         string
	Socket             string
	EnrollmentManifest string
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
	if len(value) != 36 || strings.ToLower(value) != value {
		return fmt.Errorf("TOPS ID must be a canonical lowercase UUID")
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return fmt.Errorf("TOPS ID must be a canonical lowercase UUID")
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return fmt.Errorf("TOPS ID must be a canonical lowercase UUID")
			}
		}
	}
	if value == "00000000-0000-0000-0000-000000000000" || value[14] < '1' || value[14] > '8' || !strings.Contains("89ab", value[19:20]) {
		return fmt.Errorf("TOPS ID must be a non-nil RFC UUID with version 1 through 8")
	}
	return nil
}

func ResolveInstall(scope Scope) (InstallLayout, error) {
	switch scope {
	case ScopeUser:
		home, stateBase, _, _, err := userBases()
		if err != nil {
			return InstallLayout{}, err
		}
		stateDir := filepath.Join(stateBase, "symphony", "ssiag")
		return cleanInstall(InstallLayout{
			Scope:           scope,
			Binary:          filepath.Join(home, ".local", "bin", "symphony-ssiag"),
			StateDir:        stateDir,
			InstallManifest: filepath.Join(stateDir, "install.json"),
		})
	case ScopeSystem:
		return cleanInstall(InstallLayout{
			Scope:           scope,
			Binary:          "/usr/local/bin/symphony-ssiag",
			StateDir:        "/var/lib/symphony/ssiag",
			InstallManifest: "/var/lib/symphony/ssiag/install.json",
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
		_, stateBase, configBase, runtimeBase, err := userBases()
		if err != nil {
			return InstanceLayout{}, err
		}
		configDir := filepath.Join(configBase, "symphony", topsID, "ssiag")
		stateDir := filepath.Join(stateBase, "symphony", topsID, "ssiag")
		var runtimeDir string
		if runtimeBase == "" {
			runtimeDir = filepath.Join(stateDir, "run")
		} else {
			runtimeDir = filepath.Join(runtimeBase, "symphony", topsID, "ssiag")
		}
		return cleanInstance(InstanceLayout{
			Scope:              scope,
			TOPSID:             topsID,
			ConfigDir:          configDir,
			ConfigFile:         filepath.Join(configDir, "config.json"),
			StateDir:           stateDir,
			RuntimeDir:         runtimeDir,
			Socket:             filepath.Join(runtimeDir, "ssiag.sock"),
			EnrollmentManifest: filepath.Join(stateDir, "enrollment.json"),
		})
	case ScopeSystem:
		configDir := filepath.Join("/etc/symphony", topsID, "ssiag")
		stateDir := filepath.Join("/var/lib/symphony", topsID, "ssiag")
		runtimeDir := filepath.Join(systemRuntimeRoot(), topsID, "ssiag")
		return cleanInstance(InstanceLayout{
			Scope:              scope,
			TOPSID:             topsID,
			ConfigDir:          configDir,
			ConfigFile:         filepath.Join(configDir, "config.json"),
			StateDir:           stateDir,
			RuntimeDir:         runtimeDir,
			Socket:             filepath.Join(runtimeDir, "ssiag.sock"),
			EnrollmentManifest: filepath.Join(stateDir, "enrollment.json"),
		})
	default:
		return InstanceLayout{}, fmt.Errorf("unsupported scope %q", scope)
	}
}

func systemRuntimeRoot() string {
	if runtime.GOOS == "darwin" {
		return "/var/run/symphony"
	}
	return "/run/symphony"
}

func userBases() (home, stateBase, configBase, runtimeBase string, err error) {
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", "", "", fmt.Errorf("resolve user home: %w", err)
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
	return home, stateBase, configBase, runtimeBase, nil
}

func cleanInstall(layout InstallLayout) (InstallLayout, error) {
	values := []*string{&layout.Binary, &layout.StateDir, &layout.InstallManifest}
	for _, value := range values {
		if err := cleanAbsolute(value); err != nil {
			return InstallLayout{}, err
		}
	}
	return layout, nil
}

func cleanInstance(layout InstanceLayout) (InstanceLayout, error) {
	values := []*string{&layout.ConfigDir, &layout.ConfigFile, &layout.StateDir, &layout.RuntimeDir, &layout.Socket, &layout.EnrollmentManifest}
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
