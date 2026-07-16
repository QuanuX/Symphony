package stavclient

import (
	"fmt"
	"os"
	"path/filepath"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

// SocketForTOPS resolves the ratified STAV append-authority socket namespace.
// It does not create or connect to the socket.
func SocketForTOPS(scope, topsID string) (string, error) {
	if err := stavprotocol.ValidateTOPSID(topsID); err != nil {
		return "", err
	}
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeBase != "" {
			return cleanAbsolute(filepath.Join(runtimeBase, "symphony", topsID, "stav", "append.sock"))
		}
		stateBase := os.Getenv("XDG_STATE_HOME")
		if stateBase == "" {
			stateBase = filepath.Join(home, ".local", "state")
		}
		return cleanAbsolute(filepath.Join(stateBase, "symphony", topsID, "stav", "run", "append.sock"))
	case "system":
		return filepath.Join("/run/symphony", topsID, "stav", "append.sock"), nil
	default:
		return "", fmt.Errorf("unsupported scope %q: expected user or system", scope)
	}
}

func cleanAbsolute(value string) (string, error) {
	value = filepath.Clean(value)
	if !filepath.IsAbs(value) || value == string(filepath.Separator) {
		return "", fmt.Errorf("unsafe non-absolute STAV socket path %q", value)
	}
	return value, nil
}
