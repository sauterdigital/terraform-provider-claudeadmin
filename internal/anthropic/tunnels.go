package anthropic

import (
	"context"
	"net/http"
	"net/url"
)

const tunnelsBeta = "mcp-tunnels-2026-06-22"

// All MCP Tunnel endpoints REQUIRE OAuth Bearer + the beta header.
// We attach the beta header automatically via WithBetaHeaders.

type Tunnel struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	DisplayName *string `json:"display_name"`
	Domain      string  `json:"domain"`
	WorkspaceID *string `json:"workspace_id"`
	CreatedAt   string  `json:"created_at"`
	ArchivedAt  *string `json:"archived_at"`
}

type TunnelToken struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	TunnelToken string `json:"tunnel_token"`
}

type TunnelCertificate struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	TunnelID    string  `json:"tunnel_id"`
	Fingerprint string  `json:"fingerprint"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at"`
	ArchivedAt  *string `json:"archived_at"`
}

type ListTunnelsParams struct {
	IncludeArchived bool
	WorkspaceID     string
}

type ListTunnelsResponse struct {
	Data     []Tunnel `json:"data"`
	NextPage *string  `json:"next_page"`
}

type ListTunnelCertsResponse struct {
	Data     []TunnelCertificate `json:"data"`
	NextPage *string             `json:"next_page"`
}

type CreateTunnelCertRequest struct {
	CACertificatePEM string `json:"ca_certificate_pem"`
}

type RotateTunnelTokenRequest struct {
	Reason string `json:"reason,omitempty"`
}

func tunnelsCtx(ctx context.Context) context.Context {
	return WithBetaHeaders(ctx, tunnelsBeta)
}

func (c *Client) GetTunnel(ctx context.Context, id string) (*Tunnel, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out Tunnel
	if err := c.do(tunnelsCtx(ctx), http.MethodGet, "/v1/tunnels/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListTunnels(ctx context.Context, p ListTunnelsParams) ([]Tunnel, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var all []Tunnel
	var cursor string
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("page", cursor)
		}
		if p.IncludeArchived {
			q.Set("include_archived", "true")
		}
		if p.WorkspaceID != "" {
			q.Set("workspace_id", p.WorkspaceID)
		}
		path := "/v1/tunnels"
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
		var page ListTunnelsResponse
		if err := c.do(tunnelsCtx(ctx), http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || *page.NextPage == "" {
			return all, nil
		}
		cursor = *page.NextPage
	}
}

func (c *Client) RevealTunnelToken(ctx context.Context, id string) (*TunnelToken, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out TunnelToken
	if err := c.do(tunnelsCtx(ctx), http.MethodPost, "/v1/tunnels/"+url.PathEscape(id)+"/reveal_token", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RotateTunnelToken(ctx context.Context, id, reason string) (*TunnelToken, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out TunnelToken
	body := RotateTunnelTokenRequest{Reason: reason}
	if err := c.do(tunnelsCtx(ctx), http.MethodPost, "/v1/tunnels/"+url.PathEscape(id)+"/rotate_token", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ArchiveTunnel(ctx context.Context, id string) (*Tunnel, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out Tunnel
	if err := c.do(tunnelsCtx(ctx), http.MethodPost, "/v1/tunnels/"+url.PathEscape(id)+"/archive", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Tunnel Certificates ----------

func (c *Client) CreateTunnelCertificate(ctx context.Context, tunnelID, caCertPEM string) (*TunnelCertificate, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out TunnelCertificate
	path := "/v1/tunnels/" + url.PathEscape(tunnelID) + "/certificates"
	if err := c.do(tunnelsCtx(ctx), http.MethodPost, path, CreateTunnelCertRequest{CACertificatePEM: caCertPEM}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTunnelCertificate(ctx context.Context, tunnelID, certID string) (*TunnelCertificate, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out TunnelCertificate
	path := "/v1/tunnels/" + url.PathEscape(tunnelID) + "/certificates/" + url.PathEscape(certID)
	if err := c.do(tunnelsCtx(ctx), http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListTunnelCertificates(ctx context.Context, tunnelID string, includeArchived bool) ([]TunnelCertificate, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var all []TunnelCertificate
	var cursor string
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("page", cursor)
		}
		if includeArchived {
			q.Set("include_archived", "true")
		}
		path := "/v1/tunnels/" + url.PathEscape(tunnelID) + "/certificates"
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
		var page ListTunnelCertsResponse
		if err := c.do(tunnelsCtx(ctx), http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || *page.NextPage == "" {
			return all, nil
		}
		cursor = *page.NextPage
	}
}

func (c *Client) ArchiveTunnelCertificate(ctx context.Context, tunnelID, certID string) (*TunnelCertificate, error) {
	if !c.HasOAuth() {
		return nil, ErrOAuthRequired
	}
	var out TunnelCertificate
	path := "/v1/tunnels/" + url.PathEscape(tunnelID) + "/certificates/" + url.PathEscape(certID) + "/archive"
	if err := c.do(tunnelsCtx(ctx), http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
