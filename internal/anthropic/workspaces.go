package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type Workspace struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Name          string            `json:"name"`
	CreatedAt     string            `json:"created_at"`
	ArchivedAt    *string           `json:"archived_at"`
	DisplayColor  string            `json:"display_color"`
	CompartmentID string            `json:"compartment_id"`
	ExternalKeyID string            `json:"external_key_id,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	DataResidency *DataResidency    `json:"data_residency,omitempty"`
}

// DataResidency holds the workspace data-residency block. The Admin API's
// allowed_inference_geos can be either the literal string "unrestricted" or
// an array of geo strings; we normalize both shapes onto a single []string
// where ["unrestricted"] denotes the unrestricted case.
type DataResidency struct {
	AllowedInferenceGeos AllowedInferenceGeos `json:"allowed_inference_geos,omitempty"`
	DefaultInferenceGeo  string               `json:"default_inference_geo,omitempty"`
	WorkspaceGeo         string               `json:"workspace_geo,omitempty"`
}

type AllowedInferenceGeos struct {
	Values []string
}

func (a AllowedInferenceGeos) MarshalJSON() ([]byte, error) {
	if len(a.Values) == 1 && a.Values[0] == "unrestricted" {
		return []byte(`"unrestricted"`), nil
	}
	if a.Values == nil {
		return []byte("null"), nil
	}
	return json.Marshal(a.Values)
}

func (a *AllowedInferenceGeos) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		a.Values = []string{s}
		return nil
	}
	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return fmt.Errorf("allowed_inference_geos: expected string or array, got %s", string(b))
	}
	a.Values = list
	return nil
}

type CreateWorkspaceRequest struct {
	Name          string            `json:"name"`
	Tags          map[string]string `json:"tags,omitempty"`
	ExternalKeyID string            `json:"external_key_id,omitempty"`
	DataResidency *DataResidency    `json:"data_residency,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name string            `json:"name,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type ListWorkspacesParams struct {
	BeforeID        string
	AfterID         string
	Limit           int
	IncludeArchived bool
}

type ListWorkspacesResponse struct {
	Data    []Workspace `json:"data"`
	FirstID string      `json:"first_id"`
	LastID  string      `json:"last_id"`
	HasMore bool        `json:"has_more"`
}

func (c *Client) CreateWorkspace(ctx context.Context, in CreateWorkspaceRequest) (*Workspace, error) {
	var out Workspace
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/workspaces", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	var out Workspace
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/workspaces/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateWorkspace(ctx context.Context, id string, in UpdateWorkspaceRequest) (*Workspace, error) {
	var out Workspace
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/workspaces/"+url.PathEscape(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ArchiveWorkspace(ctx context.Context, id string) (*Workspace, error) {
	var out Workspace
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/workspaces/"+url.PathEscape(id)+"/archive", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListWorkspaces(ctx context.Context, p ListWorkspacesParams) ([]Workspace, error) {
	var all []Workspace
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
		if p.IncludeArchived {
			q.Set("include_archived", "true")
		}

		var page ListWorkspacesResponse
		if err := c.do(ctx, http.MethodGet, "/v1/organizations/workspaces?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		cursor = page.LastID
	}
}
