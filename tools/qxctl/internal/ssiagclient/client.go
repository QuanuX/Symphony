package ssiagclient

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
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type Status struct {
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Ready         bool   `json:"ready"`
	Mode          string `json:"mode"`
	TOPSID        string `json:"tops_id"`
	TOPSName      string `json:"tops_name"`
	Transport     string `json:"transport"`
	ProviderCount int    `json:"provider_count"`
}

type ProviderDescriptor struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
	Exportable   bool     `json:"exportable"`
	Interactive  bool     `json:"interactive"`
}

type ProvidersResponse struct {
	Schema    string               `json:"schema"`
	Providers []ProviderDescriptor `json:"providers"`
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func SocketForTOPS(scope, topsID string) (string, error) {
	if err := validateTOPSID(topsID); err != nil {
		return "", err
	}
	if override := os.Getenv("SYMPHONY_SSIAG_SOCKET"); override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("SYMPHONY_SSIAG_SOCKET must be absolute")
		}
		return filepath.Clean(override), nil
	}
	switch scope {
	case "system":
		return filepath.Join("/run/symphony", topsID, "ssiag.sock"), nil
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeBase != "" {
			return filepath.Join(runtimeBase, "symphony", topsID, "ssiag.sock"), nil
		}
		stateBase := os.Getenv("XDG_STATE_HOME")
		if stateBase == "" {
			stateBase = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(stateBase, "symphony", topsID, "ssiag", "run", "symphony", topsID, "ssiag.sock"), nil
	default:
		return "", fmt.Errorf("unsupported SSIAG scope %q", scope)
	}
}

func validateTOPSID(value string) error {
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

func New(socket string, timeout time.Duration) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}
	return &Client{
		httpClient: &http.Client{Transport: transport, Timeout: timeout},
		baseURL:    "http://unix",
	}
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var result Status
	if err := c.get(ctx, "/v1/status", &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.status.v1" {
		return result, fmt.Errorf("unsupported SSIAG status schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) Providers(ctx context.Context) (ProvidersResponse, error) {
	var result ProvidersResponse
	if err := c.get(ctx, "/v1/providers", &result); err != nil {
		return result, err
	}
	if result.Schema != "symphony.ssiag.providers.v1" {
		return result, fmt.Errorf("unsupported SSIAG providers schema %q", result.Schema)
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
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
