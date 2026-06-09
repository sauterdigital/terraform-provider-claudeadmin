package anthropic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type WorkspaceMember struct {
	Type          string `json:"type"`
	UserID        string `json:"user_id"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceRole string `json:"workspace_role"`
}

type CreateWorkspaceMemberRequest struct {
	UserID        string `json:"user_id"`
	WorkspaceRole string `json:"workspace_role"`
}

type UpdateWorkspaceMemberRequest struct {
	WorkspaceRole string `json:"workspace_role"`
}

type ListWorkspaceMembersResponse struct {
	Data    []WorkspaceMember `json:"data"`
	FirstID string            `json:"first_id"`
	LastID  string            `json:"last_id"`
	HasMore bool              `json:"has_more"`
}

func workspaceMembersPath(workspaceID string) string {
	return "/v1/organizations/workspaces/" + url.PathEscape(workspaceID) + "/members"
}

func workspaceMemberPath(workspaceID, userID string) string {
	return workspaceMembersPath(workspaceID) + "/" + url.PathEscape(userID)
}

func (c *Client) AddWorkspaceMember(ctx context.Context, workspaceID string, in CreateWorkspaceMemberRequest) (*WorkspaceMember, error) {
	var out WorkspaceMember
	if err := c.do(ctx, http.MethodPost, workspaceMembersPath(workspaceID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetWorkspaceMember(ctx context.Context, workspaceID, userID string) (*WorkspaceMember, error) {
	var out WorkspaceMember
	if err := c.do(ctx, http.MethodGet, workspaceMemberPath(workspaceID, userID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateWorkspaceMember(ctx context.Context, workspaceID, userID string, in UpdateWorkspaceMemberRequest) (*WorkspaceMember, error) {
	var out WorkspaceMember
	if err := c.do(ctx, http.MethodPost, workspaceMemberPath(workspaceID, userID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteWorkspaceMember(ctx context.Context, workspaceID, userID string) error {
	return c.do(ctx, http.MethodDelete, workspaceMemberPath(workspaceID, userID), nil, nil)
}

func (c *Client) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]WorkspaceMember, error) {
	var all []WorkspaceMember
	var cursor string
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("after_id", cursor)
		}
		q.Set("limit", strconv.Itoa(1000))

		var page ListWorkspaceMembersResponse
		path := workspaceMembersPath(workspaceID) + "?" + q.Encode()
		if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		cursor = page.LastID
	}
}
