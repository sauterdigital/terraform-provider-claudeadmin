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

// Organization-level rate limits use the same group shape as workspace rate
// limits, but the per-limit value is just {type, value} — no org_limit field
// because these ARE the org limits.

type OrganizationRateLimitGroup struct {
	Type      string              `json:"type"`
	GroupType string              `json:"group_type"`
	Models    []string            `json:"models"`
	Limits    []OrgRateLimitValue `json:"limits"`
}

type OrgRateLimitValue struct {
	Type  string `json:"type"`
	Value int64  `json:"value"`
}

type ListOrganizationRateLimitsParams struct {
	GroupType string
	Model     string
}

type ListOrganizationRateLimitsResponse struct {
	Data     []OrganizationRateLimitGroup `json:"data"`
	NextPage *string                      `json:"next_page"`
}

// ListOrganizationRateLimits returns the Messages API rate limits configured
// at the organization level. Each entry corresponds to one rate-limit group
// (model family, batch, token_count, files, skills, web_search). Unlike the
// workspace endpoint, every group active on the org is returned (not just
// overrides), so this is the source of truth for what the org actually has.
func (c *Client) ListOrganizationRateLimits(ctx context.Context, p ListOrganizationRateLimitsParams) ([]OrganizationRateLimitGroup, error) {
	var all []OrganizationRateLimitGroup
	var cursor string
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("page", cursor)
		}
		if p.GroupType != "" {
			q.Set("group_type", p.GroupType)
		}
		if p.Model != "" {
			q.Set("model", p.Model)
		}
		path := "/v1/organizations/rate_limits"
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}

		var page ListOrganizationRateLimitsResponse
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
