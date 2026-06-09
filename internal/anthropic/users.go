package anthropic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type User struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	AddedAt string `json:"added_at"`
}

type UpdateUserRequest struct {
	Role string `json:"role"`
}

type ListUsersParams struct {
	BeforeID string
	AfterID  string
	Limit    int
	Email    string
}

type ListUsersResponse struct {
	Data    []User `json:"data"`
	FirstID string `json:"first_id"`
	LastID  string `json:"last_id"`
	HasMore bool   `json:"has_more"`
}

func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/users/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateUser(ctx context.Context, id string, in UpdateUserRequest) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/users/"+url.PathEscape(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/organizations/users/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListUsers(ctx context.Context, p ListUsersParams) ([]User, error) {
	var all []User
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
		if p.Email != "" {
			q.Set("email", p.Email)
		}

		var page ListUsersResponse
		if err := c.do(ctx, http.MethodGet, "/v1/organizations/users?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		cursor = page.LastID
	}
}
