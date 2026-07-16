package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/peerauth"
)

const defaultDeadline = 5 * time.Second

type Client struct {
	config   stavprotocol.AppendAuthorityConfig
	resolver peerauth.Resolver
}

// LoadConfig applies the append authority's strict, bounded configuration
// parser without exposing its internal implementation package.
func LoadConfig(path string) (stavprotocol.AppendAuthorityConfig, error) {
	return config.Load(path)
}

func New(cfg stavprotocol.AppendAuthorityConfig) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	resolver, err := peerauth.NewResolver(cfg.Authentication)
	if err != nil {
		return nil, err
	}
	return &Client{config: cfg, resolver: resolver}, nil
}

func (c *Client) Do(ctx context.Context, request stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	if err := request.Validate(); err != nil {
		return stavprotocol.LocalResponse{}, err
	}
	if request.TOPSID != c.config.TOPSID {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: request TOPS does not match configuration")
	}
	info, err := os.Lstat(c.config.Listen.Address)
	if err != nil {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: inspect socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: endpoint is not a Unix socket")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", c.config.Listen.Address)
	if err != nil {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: connect: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(defaultDeadline)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: set deadline: %w", err)
	}
	if err := c.resolver.VerifyAuthority(conn); err != nil {
		return stavprotocol.LocalResponse{}, err
	}
	payload, err := stavprotocol.EncodeLocalRequest(request)
	if err != nil {
		return stavprotocol.LocalResponse{}, err
	}
	if err := stavprotocol.WriteFrame(conn, payload, stavprotocol.MaxRequestBytes); err != nil {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: write request: %w", err)
	}
	responsePayload, err := stavprotocol.ReadFrame(conn, stavprotocol.MaxResponseBytes)
	if err != nil {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: read response: %w", err)
	}
	response, err := stavprotocol.DecodeLocalResponse(responsePayload)
	if err != nil {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: decode response: %w", err)
	}
	if response.RequestID != request.RequestID || response.Operation != request.Operation || response.TOPSID != request.TOPSID {
		return stavprotocol.LocalResponse{}, fmt.Errorf("stav client: response binding mismatch")
	}
	return response, nil
}
