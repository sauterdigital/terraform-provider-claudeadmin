package anthropic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type ExternalKey struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	DisplayName    string         `json:"display_name"`
	Geo            string         `json:"geo"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	ProviderConfig ProviderConfig `json:"provider_config"`
}

// ProviderConfig is the polymorphic CMEK provider config. The Admin API
// expects a flat object discriminated by `type` (aws / gcp / azure). We model
// it with all possible fields and `omitempty` so only the relevant subset
// is serialized for each type.
type ProviderConfig struct {
	Type string `json:"type"`

	// AWS
	KMSArn  string `json:"kms_arn,omitempty"`
	RoleArn string `json:"role_arn,omitempty"`
	Region  string `json:"region,omitempty"`

	// GCP + Azure share key_name
	KeyName string `json:"key_name,omitempty"`

	// Azure
	TenantID string `json:"tenant_id,omitempty"`
	VaultURI string `json:"vault_uri,omitempty"`
	ClientID string `json:"client_id,omitempty"`
}

type CreateExternalKeyRequest struct {
	DisplayName    string         `json:"display_name"`
	ProviderConfig ProviderConfig `json:"provider_config"`
	Geo            string         `json:"geo,omitempty"`
}

type UpdateExternalKeyRequest struct {
	DisplayName    *string         `json:"display_name,omitempty"`
	Geo            *string         `json:"geo,omitempty"`
	ProviderConfig *ProviderConfig `json:"provider_config,omitempty"`
}

type ListExternalKeysResponse struct {
	Data     []ExternalKey `json:"data"`
	NextPage *string       `json:"next_page"`
}

type ExternalKeyValidation struct {
	Status string  `json:"status"`
	Error  *string `json:"error"`
	Type   string  `json:"type"`
}

func (c *Client) CreateExternalKey(ctx context.Context, in CreateExternalKeyRequest) (*ExternalKey, error) {
	var out ExternalKey
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/external_keys", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetExternalKey(ctx context.Context, id string) (*ExternalKey, error) {
	var out ExternalKey
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/external_keys/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateExternalKey(ctx context.Context, id string, in UpdateExternalKeyRequest) (*ExternalKey, error) {
	var out ExternalKey
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/external_keys/"+url.PathEscape(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteExternalKey(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/organizations/external_keys/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ValidateExternalKey(ctx context.Context, id string) (*ExternalKeyValidation, error) {
	var out ExternalKeyValidation
	if err := c.do(ctx, http.MethodPost, "/v1/organizations/external_keys/"+url.PathEscape(id)+"/validate", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListExternalKeys(ctx context.Context) ([]ExternalKey, error) {
	var all []ExternalKey
	var cursor string
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(100))
		if cursor != "" {
			q.Set("page", cursor)
		}
		var page ListExternalKeysResponse
		if err := c.do(ctx, http.MethodGet, "/v1/organizations/external_keys?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || *page.NextPage == "" {
			return all, nil
		}
		cursor = *page.NextPage
	}
}
