package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
)

const maxResponseBytes = 1 << 20

type Client struct {
	httpClient *http.Client
}

func New(socket string, timeout time.Duration) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}
	return &Client{httpClient: &http.Client{Transport: transport, Timeout: timeout}}
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
