//go:build darwin || linux

package foundationaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	stav "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/peerauth"
)

// SubmitToOwner selects SSIAG solely from the scoped canonical TOPS layout.
// Its small configuration projection is not a general SSIAG config writer.
func SubmitToOwner(ctx context.Context, request Request) (Response, error) {
	var response Response
	if err := request.Validate(); err != nil {
		return response, err
	}
	configFile, socket, err := ssiagPaths(request.Scope, request.TOPSID)
	if err != nil {
		return response, err
	}
	fd, err := syscall.Open(configFile, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return response, err
	}
	f := os.NewFile(uintptr(fd), configFile)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return response, err
	}
	raw, ok := st.Sys().(*syscall.Stat_t)
	owner := uint32(os.Geteuid())
	mask := os.FileMode(0077)
	if request.Scope == "system" {
		owner = 0
		mask = 0022
	}
	if !ok || !st.Mode().IsRegular() || raw.Uid != owner || st.Mode().Perm()&mask != 0 || st.Size() > 1<<20 {
		return response, fmt.Errorf("untrusted SSIAG endpoint configuration")
	}
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return response, err
	}
	data, err = stav.Canonicalize(data)
	if err != nil {
		return response, err
	}
	var cfg struct {
		Schema string `json:"schema"`
		Mode   string `json:"mode"`
		TOPS   struct {
			ID string `json:"id"`
		} `json:"tops"`
		Listen struct {
			Network string `json:"network"`
			Address string `json:"address"`
		} `json:"listen"`
		Authentication struct {
			Mechanism string `json:"mechanism"`
			Service   *struct {
				ID   string  `json:"id"`
				Kind string  `json:"kind"`
				UID  *uint32 `json:"uid"`
				GID  *uint32 `json:"gid"`
			} `json:"service"`
		} `json:"authentication"`
	}
	if json.Unmarshal(data, &cfg) != nil || cfg.Schema != "symphony.ssiag.config.v1" || cfg.Mode != request.Scope || cfg.TOPS.ID != request.TOPSID || cfg.Listen.Network != "unix" || cfg.Listen.Address != socket || cfg.Authentication.Mechanism != "unix_peer_credentials" || cfg.Authentication.Service == nil || cfg.Authentication.Service.UID == nil || cfg.Authentication.Service.GID == nil || cfg.Authentication.Service.ID != "symphony.ssiag.service" || cfg.Authentication.Service.Kind != "symphony.identity.service" {
		return response, fmt.Errorf("SSIAG endpoint identity binding mismatch")
	}
	uid, gid := *cfg.Authentication.Service.UID, *cfg.Authentication.Service.GID
	if uid == ^uint32(0) || gid == ^uint32(0) {
		return response, fmt.Errorf("invalid SSIAG endpoint identity")
	}
	if override := os.Getenv("SYMPHONY_SSIAG_SOCKET"); override != "" {
		if !filepath.IsAbs(override) {
			return response, fmt.Errorf("SSIAG socket override must be absolute")
		}
		socket = filepath.Clean(override)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		info, err := os.Lstat(socket)
		if err != nil {
			return nil, err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || info.Mode()&os.ModeSocket == 0 || stat.Uid != uid {
			return nil, fmt.Errorf("SSIAG endpoint is not the configured owned socket")
		}
		conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
		if err != nil {
			return nil, err
		}
		peer, err := peerauth.CredentialsFromConn(conn)
		if err != nil || peer.UID != uid || peer.GID != gid {
			conn.Close()
			return nil, fmt.Errorf("SSIAG endpoint kernel identity mismatch")
		}
		return conn, nil
	}}
	defer transport.CloseIdleConnections()
	body, err := json.Marshal(request)
	if err != nil {
		return response, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+Endpoint, bytes.NewReader(body))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return response, fmt.Errorf("SSIAG foundational audit returned HTTP %d", res.StatusCode)
	}
	data, err = io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil {
		return response, err
	}
	if err = Decode(data, &response); err != nil {
		return response, err
	}
	return response, response.Validate(request, uint32(os.Geteuid()))
}

func ssiagPaths(scope, tops string) (string, string, error) {
	if err := stav.ValidateTOPSID(tops); err != nil {
		return "", "", err
	}
	var configBase, stateBase, runtimeBase string
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
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
			runtimeBase = filepath.Join(stateBase, "symphony", tops, "ssiag", "run")
		} else {
			runtimeBase = filepath.Join(runtimeBase, "symphony", tops, "ssiag")
		}
	case "system":
		configBase = "/etc"
		runtimeBase = "/run/symphony"
		if runtime.GOOS == "darwin" {
			runtimeBase = "/var/run/symphony"
		}
		runtimeBase = filepath.Join(runtimeBase, tops, "ssiag")
	default:
		return "", "", fmt.Errorf("invalid scope")
	}
	configFile := filepath.Join(configBase, "symphony", tops, "ssiag", "config.json")
	if !filepath.IsAbs(configFile) || !filepath.IsAbs(runtimeBase) {
		return "", "", fmt.Errorf("SSIAG selected paths must be absolute")
	}
	return configFile, filepath.Join(runtimeBase, "ssiag.sock"), nil
}
