package anthropic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type Invite struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	InvitedAt string `json:"invited_at"`
	ExpiresAt string `json:"expires_at"`
}

type CreateInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ListInvitesResponse struct {
	Data    []Invite `json:"data"`
	FirstID string   `json:"first_id"`
	LastID  string   `json:"last_id"`
	HasMore bool     `json:"has_more"`
}

func (c *Client) CreateInvite(ctx context.Context, in CreateInviteRequest) (*Invite, error) {
	var out Invite
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/invites", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetInvite(ctx context.Context, id string) (*Invite, error) {
	var out Invite
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/invites/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteInvite(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/organizations/invites/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListInvites(ctx context.Context) ([]Invite, error) {
	var all []Invite
	var cursor string
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("after_id", cursor)
		}
		q.Set("limit", strconv.Itoa(1000))

		var page ListInvitesResponse
		if err := c.do(ctx, http.MethodGet, "/v1/organizations/invites?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		cursor = page.LastID
	}
}
