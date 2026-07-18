package config

import (
	"fmt"
	"os"

	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

// LoadTrusted binds client and server endpoint trust to the configuration file
// selected by an immutable TOPS ID and installation scope.
func LoadTrusted(path string, scope ssiagpaths.Scope) (Config, error) {
	file, err := openNoFollow(path)
	if err != nil {
		return Config{}, fmt.Errorf("open trusted SSIAG configuration: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Config{}, fmt.Errorf("stat trusted SSIAG configuration: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Config{}, fmt.Errorf("trusted SSIAG configuration is not a regular file")
	}
	owner, err := fileOwnerUID(info)
	if err != nil {
		return Config{}, err
	}
	switch scope {
	case ssiagpaths.ScopeUser:
		if owner != uint32(os.Geteuid()) {
			return Config{}, fmt.Errorf("user SSIAG configuration owner uid=%d does not match effective uid=%d", owner, os.Geteuid())
		}
		if info.Mode().Perm()&0o077 != 0 {
			return Config{}, fmt.Errorf("user SSIAG configuration must be owner-only")
		}
	case ssiagpaths.ScopeSystem:
		if owner != 0 {
			return Config{}, fmt.Errorf("system SSIAG configuration must be administrator-owned")
		}
		if info.Mode().Perm()&0o022 != 0 {
			return Config{}, fmt.Errorf("system SSIAG configuration must not be writable by group or other")
		}
	default:
		return Config{}, fmt.Errorf("unsupported SSIAG trust scope %q", scope)
	}
	return decode(file)
}
