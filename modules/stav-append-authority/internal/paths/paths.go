package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

const BinaryName = "symphony-stav-append-authority"

// InstallLayout contains the independently installed host executable.
type InstallLayout struct {
	Scope          Scope
	Prefix         string
	Binary         string
	Manifest       string
	LegacyBinary   string
	LegacyManifest string
}

// InstanceLayout resolves the Architect-ratified per-TOPS namespace.
type InstanceLayout struct {
	Scope          Scope
	TOPSID         string
	ConfigDir      string
	ConfigFile     string
	EnrollmentFile string
	StateDir       string
	LedgerFile     string
	RecoveryDir    string
	RuntimeDir     string
	Socket         string
	LifecycleDir   string
	LifecycleLock  string
	ActiveAttempt  string
	ActivePlan     string
	LastAttempt    string
	AttemptDir     string
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
		return versionedInstall(scope, filepath.Join(home, ".local"))
	case ScopeSystem:
		return versionedInstall(scope, "/usr/local")
	default:
		return InstallLayout{}, fmt.Errorf("unsupported scope %q", scope)
	}
}

func ResolveInstallAt(scope Scope, prefix, requestedVersion string) (InstallLayout, error) {
	if requestedVersion != version.Version {
		return InstallLayout{}, fmt.Errorf("requested STAV package version %q does not match compiled version %q", requestedVersion, version.Version)
	}
	if scope != ScopeUser && scope != ScopeSystem {
		return InstallLayout{}, fmt.Errorf("unsupported scope %q", scope)
	}
	return versionedInstall(scope, prefix)
}

func versionedInstall(scope Scope, prefix string) (InstallLayout, error) {
	if version.Version == "" {
		return InstallLayout{}, fmt.Errorf("empty STAV build version")
	}
	for _, character := range version.Version {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '.' || character == '+' || character == '-') {
			return InstallLayout{}, fmt.Errorf("unsafe STAV build version %q", version.Version)
		}
	}
	return cleanInstall(InstallLayout{
		Scope: scope, Prefix: prefix,
		Binary:         filepath.Join(prefix, "libexec", "symphony", "stav-append-authority", version.Version, BinaryName),
		Manifest:       filepath.Join(prefix, "share", "symphony", "receipts", "stav-append-authority", version.Version, "install-receipt.json"),
		LegacyBinary:   filepath.Join(prefix, "bin", BinaryName),
		LegacyManifest: filepath.Join(prefix, "bin", BinaryName+".install.json"),
	})
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
		lifecycleDir := filepath.Join(stateBase, "symphony", topsID, "foundation-lifecycle", "stav")
		runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
		var runtimeDir string
		if runtimeBase == "" {
			runtimeDir = filepath.Join(stateDir, "run")
		} else {
			runtimeDir = filepath.Join(runtimeBase, "symphony", topsID, "stav")
		}
		return cleanInstance(InstanceLayout{
			Scope:          scope,
			TOPSID:         topsID,
			ConfigDir:      configDir,
			ConfigFile:     filepath.Join(configDir, "append-authority.json"),
			EnrollmentFile: filepath.Join(configDir, "enrollment.json"),
			StateDir:       stateDir,
			LedgerFile:     filepath.Join(stateDir, "ledger-v1.stavlog"),
			RecoveryDir:    filepath.Join(stateDir, "recovery"),
			RuntimeDir:     runtimeDir,
			Socket:         filepath.Join(runtimeDir, "append.sock"),
			LifecycleDir:   lifecycleDir,
			LifecycleLock:  filepath.Join(lifecycleDir, "lifecycle.lock"),
			ActiveAttempt:  filepath.Join(lifecycleDir, "active-attempt.json"),
			ActivePlan:     filepath.Join(lifecycleDir, "active-plan.json"),
			LastAttempt:    filepath.Join(lifecycleDir, "last-attempt.json"),
			AttemptDir:     filepath.Join(lifecycleDir, "attempts"),
		})
	case ScopeSystem:
		configDir := filepath.Join("/etc/symphony", topsID, "stav")
		stateDir := filepath.Join("/var/lib/symphony", topsID, "stav")
		lifecycleDir := filepath.Join("/var/lib/symphony", topsID, "foundation-lifecycle", "stav")
		runtimeDir := filepath.Join(systemRuntimeRoot(), topsID, "stav")
		return cleanInstance(InstanceLayout{
			Scope:          scope,
			TOPSID:         topsID,
			ConfigDir:      configDir,
			ConfigFile:     filepath.Join(configDir, "append-authority.json"),
			EnrollmentFile: filepath.Join(configDir, "enrollment.json"),
			StateDir:       stateDir,
			LedgerFile:     filepath.Join(stateDir, "ledger-v1.stavlog"),
			RecoveryDir:    filepath.Join(stateDir, "recovery"),
			RuntimeDir:     runtimeDir,
			Socket:         filepath.Join(runtimeDir, "append.sock"),
			LifecycleDir:   lifecycleDir,
			LifecycleLock:  filepath.Join(lifecycleDir, "lifecycle.lock"),
			ActiveAttempt:  filepath.Join(lifecycleDir, "active-attempt.json"),
			ActivePlan:     filepath.Join(lifecycleDir, "active-plan.json"),
			LastAttempt:    filepath.Join(lifecycleDir, "last-attempt.json"),
			AttemptDir:     filepath.Join(lifecycleDir, "attempts"),
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

func cleanInstall(layout InstallLayout) (InstallLayout, error) {
	for _, value := range []*string{&layout.Prefix, &layout.Binary, &layout.Manifest, &layout.LegacyBinary, &layout.LegacyManifest} {
		if err := cleanAbsolute(value); err != nil {
			return InstallLayout{}, err
		}
	}
	return layout, nil
}

func cleanInstance(layout InstanceLayout) (InstanceLayout, error) {
	values := []*string{&layout.ConfigDir, &layout.ConfigFile, &layout.EnrollmentFile, &layout.StateDir, &layout.LedgerFile, &layout.RecoveryDir, &layout.RuntimeDir, &layout.Socket, &layout.LifecycleDir, &layout.LifecycleLock, &layout.ActiveAttempt, &layout.ActivePlan, &layout.LastAttempt, &layout.AttemptDir}
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
