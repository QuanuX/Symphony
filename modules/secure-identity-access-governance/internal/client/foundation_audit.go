package client

import (
	"context"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationaudit"
	"os"
)

func (c *Client) FoundationAudit(ctx context.Context, request foundationaudit.Request) (foundationaudit.Response, error) {
	var response foundationaudit.Response
	if err := request.Validate(); err != nil {
		return response, err
	}
	if err := c.post(ctx, foundationaudit.Endpoint, request, &response); err != nil {
		return response, err
	}
	return response, response.Validate(request, uint32(os.Geteuid()))
}
