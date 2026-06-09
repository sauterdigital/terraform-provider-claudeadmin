package anthropic

import (
	"context"
	"net/http"
	"net/url"
)

type WorkspaceRateLimitGroup struct {
	Type      string           `json:"type"`
	GroupType string           `json:"group_type"`
	Models    []string         `json:"models"`
	Limits    []RateLimitValue `json:"limits"`
}

type RateLimitValue struct {
	Type     string `json:"type"`
	Value    int64  `json:"value"`
	OrgLimit *int64 `json:"org_limit"`
}

type ListWorkspaceRateLimitsResponse struct {
	Data     []WorkspaceRateLimitGroup `json:"data"`
	NextPage *string                   `json:"next_page"`
}

// ListWorkspaceRateLimits returns the rate-limit overrides configured for a
// workspace. Groups without an override are inherited from the organization
// and do NOT appear in the response — the absence of a group means "inherit",
// not "no limit".
func (c *Client) ListWorkspaceRateLimits(ctx context.Context, workspaceID, groupType string) ([]WorkspaceRateLimitGroup, error) {
	var all []WorkspaceRateLimitGroup
	var cursor string
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("page", cursor)
		}
		if groupType != "" {
			q.Set("group_type", groupType)
		}
		path := "/v1/organizations/workspaces/" + url.PathEscape(workspaceID) + "/rate_limits"
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}

		var page ListWorkspaceRateLimitsResponse
		if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || *page.NextPage == "" {
			return all, nil
		}
		cursor = *page.NextPage
	}
}
