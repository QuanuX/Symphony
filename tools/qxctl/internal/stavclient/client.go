package stavclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	appendclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
)

type Client struct {
	inner *appendclient.Client
}

func NewForTOPS(scope, topsID string) (*Client, error) {
	path, err := ConfigForTOPS(scope, topsID)
	if err != nil {
		return nil, err
	}
	if override := os.Getenv("SYMPHONY_STAV_CONFIG"); override != "" {
		if !filepath.IsAbs(override) {
			return nil, fmt.Errorf("SYMPHONY_STAV_CONFIG must be absolute")
		}
		path = filepath.Clean(override)
	}
	cfg, err := appendclient.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	if cfg.TOPSID != topsID || cfg.Mode != scope {
		return nil, fmt.Errorf("STAV configuration does not match requested TOPS and scope")
	}
	inner, err := appendclient.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{inner: inner}, nil
}

func (c *Client) Do(ctx context.Context, request stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error) {
	return c.inner.Do(ctx, request)
}
