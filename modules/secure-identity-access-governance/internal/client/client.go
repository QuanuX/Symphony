package client

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
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/peerauth"
)

const maxResponseBytes = 1 << 20

type Client struct {
	httpClient *http.Client
}

type peerVerifier func(net.Conn, uint32, uint32) error

func NewForTOPS(scope ssiagpaths.Scope, topsID string, timeout time.Duration) (*Client, error) {
	return newForTOPS(scope, topsID, timeout, verifyPeer)
}

func newForTOPS(scope ssiagpaths.Scope, topsID string, timeout time.Duration, verifier peerVerifier) (*Client, error) {
	if verifier == nil {
		return nil, fmt.Errorf("SSIAG endpoint verifier is required")
	}
	layout, err := ssiagpaths.ResolveInstance(scope, topsID)
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadTrusted(layout.ConfigFile, scope)
	if err != nil {
		return nil, fmt.Errorf("load SSIAG configuration: %w", err)
	}
	if cfg.TOPS.ID != topsID || cfg.Mode != string(scope) {
		return nil, fmt.Errorf("SSIAG configuration does not match requested TOPS and scope")
	}
	if cfg.Listen.Address != layout.Socket {
		return nil, fmt.Errorf("SSIAG configuration socket does not match requested TOPS layout")
	}
	if cfg.Authentication == nil || cfg.Authentication.Service == nil {
		return nil, fmt.Errorf("SSIAG configuration lacks canonical service identity")
	}

	expectedUID := *cfg.Authentication.Service.UID
	expectedGID := *cfg.Authentication.Service.GID

	socket := cfg.Listen.Address
	if override := os.Getenv("SYMPHONY_SSIAG_SOCKET"); override != "" {
		if !filepath.IsAbs(override) {
			return nil, fmt.Errorf("SYMPHONY_SSIAG_SOCKET must be absolute")
		}
		socket = filepath.Clean(override)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			info, err := os.Lstat(socket)
			if err != nil {
				return nil, fmt.Errorf("inspect SSIAG socket: %w", err)
			}
			if info.Mode()&os.ModeSocket == 0 {
				return nil, fmt.Errorf("SSIAG endpoint is not a Unix socket")
			}
			owner, err := socketOwnerUID(info)
			if err != nil {
				return nil, err
			}
			if owner != expectedUID {
				return nil, fmt.Errorf("SSIAG socket owner uid=%d does not match configured service uid=%d", owner, expectedUID)
			}
			conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
			if err != nil {
				return nil, err
			}
			if err := verifier(conn, expectedUID, expectedGID); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return conn, nil
		},
	}

	return &Client{httpClient: &http.Client{Transport: transport, Timeout: timeout}}, nil
}

func verifyPeer(conn net.Conn, expectedUID, expectedGID uint32) error {
	credentials, err := peerauth.CredentialsFromConn(conn)
	if err != nil {
		return fmt.Errorf("extract SSIAG peer credentials: %w", err)
	}
	if credentials.UID != expectedUID || credentials.GID != expectedGID {
		return fmt.Errorf("SSIAG peer identity uid=%d gid=%d does not match configured service uid=%d gid=%d", credentials.UID, credentials.GID, expectedUID, expectedGID)
	}
	return nil
}

func (c *Client) Status(ctx context.Context) (model.Status, error) {
	var result model.Status
	if err := c.get(ctx, "/v1/status", &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.status.v1" {
		return result, fmt.Errorf("unsupported SSIAG status schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) Providers(ctx context.Context) (model.ProvidersResponse, error) {
	var result model.ProvidersResponse
	if err := c.get(ctx, "/v1/providers", &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.providers.v1" {
		return result, fmt.Errorf("unsupported SSIAG providers schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("SSIAG request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("SSIAG returned HTTP %d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read SSIAG response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return fmt.Errorf("SSIAG response exceeds %d bytes", maxResponseBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode SSIAG response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode SSIAG response: multiple JSON values")
		}
		return fmt.Errorf("decode trailing SSIAG response: %w", err)
	}
	return nil
}
