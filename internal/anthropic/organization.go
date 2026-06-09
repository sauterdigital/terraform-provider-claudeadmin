package anthropic

import (
	"context"
	"net/http"
)

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *Client) GetCurrentOrganization(ctx context.Context) (*Organization, error) {
	var out Organization
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/me", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
