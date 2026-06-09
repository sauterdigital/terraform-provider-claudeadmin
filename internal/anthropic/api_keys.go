package anthropic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type APIKey struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	CreatedAt      string  `json:"created_at"`
	CreatedBy      Actor   `json:"created_by"`
	ExpiresAt      *string `json:"expires_at"`
	PartialKeyHint string  `json:"partial_key_hint"`
	Status         string  `json:"status"`
	WorkspaceID    *string `json:"workspace_id"`
}

type Actor struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type UpdateAPIKeyRequest struct {
	Name   *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

type ListAPIKeysParams struct {
	BeforeID        string
	AfterID         string
	Limit           int
	Status          string
	WorkspaceID     string
	CreatedByUserID string
}

type ListAPIKeysResponse struct {
	Data    []APIKey `json:"data"`
	FirstID string   `json:"first_id"`
	LastID  string   `json:"last_id"`
	HasMore bool     `json:"has_more"`
}

func (c *Client) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	var out APIKey
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/api_keys/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateAPIKey(ctx context.Context, id string, in UpdateAPIKeyRequest) (*APIKey, error) {
	var out APIKey
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/api_keys/"+url.PathEscape(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListAPIKeys(ctx context.Context, p ListAPIKeysParams) ([]APIKey, error) {
	var all []APIKey
	cursor := p.AfterID
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("after_id", cursor)
		} else if p.BeforeID != "" {
			q.Set("before_id", p.BeforeID)
		}
		limit := p.Limit
		if limit <= 0 {
			limit = 1000
		}
		q.Set("limit", strconv.Itoa(limit))
		if p.Status != "" {
			q.Set("status", p.Status)
		}
		if p.WorkspaceID != "" {
			q.Set("workspace_id", p.WorkspaceID)
		}
		if p.CreatedByUserID != "" {
			q.Set("created_by_user_id", p.CreatedByUserID)
		}

		var page ListAPIKeysResponse
		if err := c.do(ctx, http.MethodGet, "/v1/organizations/api_keys?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		cursor = page.LastID
	}
}
