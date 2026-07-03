package anthropic

import (
	"context"
	"net/http"
	"net/url"
)

// Compliance API surface (/v1/compliance/*). READ-ONLY. Requires the
// dedicated Compliance API key (sk-ant-api01-...) — Admin key and OAuth
// bearer are rejected.
//
// Scopes on the compliance key gate individual endpoints:
//   read:compliance_activities  — activity feed only
//   read:compliance_org_data    — organizations, roles, groups, settings
//   read:compliance_user_data   — org users, group members

// ---- Types ----

type ComplianceActivity struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Timestamp    string                 `json:"timestamp"`
	ActorID      *string                `json:"actor_id"`
	ActorType    *string                `json:"actor_type"`
	ActorEmail   *string                `json:"actor_email"`
	Action       string                 `json:"action"`
	ResourceID   *string                `json:"resource_id"`
	ResourceType *string                `json:"resource_type"`
	Outcome      *string                `json:"outcome"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type ComplianceOrganization struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	DisplayName string `json:"display_name"`
	Plan        string `json:"plan"`
}

type ComplianceUser struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	LastActiveAt *string `json:"last_active_at"`
	SourceType   *string `json:"source_type"`
	ExternalID   *string `json:"external_id"`
}

type CompliancePermission struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
	Effect   string `json:"effect"`
}

type ComplianceRole struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	IsBuiltIn   bool                   `json:"is_built_in"`
	Permissions []CompliancePermission `json:"permissions"`
}

type ComplianceGroup struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	DisplayName *string `json:"display_name"`
	SourceType  *string `json:"source_type"` // "scim" | "native"
	ExternalID  *string `json:"external_id"`
	MemberCount *int64  `json:"member_count"`
}

type ComplianceGroupMember struct {
	UserID     string  `json:"user_id"`
	Email      *string `json:"email"`
	Name       *string `json:"name"`
	JoinedAt   *string `json:"joined_at"`
	SourceType *string `json:"source_type"`
}

type ComplianceOrgSettings struct {
	OrganizationID        string                 `json:"organization_id"`
	Type                  string                 `json:"type"`
	SSOEnforced           *bool                  `json:"sso_enforced"`
	MFAEnforced           *bool                  `json:"mfa_enforced"`
	SCIMEnabled           *bool                  `json:"scim_enabled"`
	AuditLogRetentionDays *int64                 `json:"audit_log_retention_days"`
	NetworkACLEnabled     *bool                  `json:"network_acl_enabled"`
	DataResidency         *string                `json:"data_residency"`
	Extra                 map[string]interface{} `json:"extra"`
}

// ---- Params ----

type ListComplianceActivitiesParams struct {
	StartingAt   string
	EndingAt     string
	ActorID      string
	Action       string
	ResourceType string
	Limit        int
}

type paginatedComplianceActivities struct {
	Data    []ComplianceActivity `json:"data"`
	HasMore bool                 `json:"has_more"`
	LastID  string               `json:"last_id"`
}

type paginatedComplianceOrgs struct {
	Data    []ComplianceOrganization `json:"data"`
	HasMore bool                     `json:"has_more"`
	LastID  string                   `json:"last_id"`
}

type paginatedComplianceUsers struct {
	Data    []ComplianceUser `json:"data"`
	HasMore bool             `json:"has_more"`
	LastID  string           `json:"last_id"`
}

type paginatedComplianceRoles struct {
	Data    []ComplianceRole `json:"data"`
	HasMore bool             `json:"has_more"`
	LastID  string           `json:"last_id"`
}

type paginatedComplianceGroups struct {
	Data    []ComplianceGroup `json:"data"`
	HasMore bool              `json:"has_more"`
	LastID  string            `json:"last_id"`
}

type paginatedComplianceGroupMembers struct {
	Data    []ComplianceGroupMember `json:"data"`
	HasMore bool                    `json:"has_more"`
	LastID  string                  `json:"last_id"`
}

// ---- Helpers ----

func complianceCtx(ctx context.Context) context.Context {
	return WithComplianceAuth(ctx)
}

func (c *Client) requireCompliance() error {
	if !c.HasCompliance() {
		return ErrComplianceRequired
	}
	return nil
}

// ---- Endpoints ----

func (c *Client) ListComplianceActivities(ctx context.Context, p ListComplianceActivitiesParams) ([]ComplianceActivity, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var all []ComplianceActivity
	var afterID string
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	for {
		q := url.Values{}
		q.Set("limit", intStr(limit))
		if afterID != "" {
			q.Set("after_id", afterID)
		}
		if p.StartingAt != "" {
			q.Set("starting_at", p.StartingAt)
		}
		if p.EndingAt != "" {
			q.Set("ending_at", p.EndingAt)
		}
		if p.ActorID != "" {
			q.Set("actor_id", p.ActorID)
		}
		if p.Action != "" {
			q.Set("action", p.Action)
		}
		if p.ResourceType != "" {
			q.Set("resource_type", p.ResourceType)
		}
		var page paginatedComplianceActivities
		if err := c.do(complianceCtx(ctx), http.MethodGet, "/v1/compliance/activities?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		afterID = page.LastID
	}
}

func (c *Client) ListComplianceOrganizations(ctx context.Context) ([]ComplianceOrganization, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var all []ComplianceOrganization
	var afterID string
	for {
		q := url.Values{}
		q.Set("limit", "100")
		if afterID != "" {
			q.Set("after_id", afterID)
		}
		var page paginatedComplianceOrgs
		if err := c.do(complianceCtx(ctx), http.MethodGet, "/v1/compliance/organizations?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		afterID = page.LastID
	}
}

func (c *Client) ListComplianceOrganizationUsers(ctx context.Context, orgID string) ([]ComplianceUser, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var all []ComplianceUser
	var afterID string
	for {
		q := url.Values{}
		q.Set("limit", "100")
		if afterID != "" {
			q.Set("after_id", afterID)
		}
		var page paginatedComplianceUsers
		path := "/v1/compliance/organizations/" + url.PathEscape(orgID) + "/users?" + q.Encode()
		if err := c.do(complianceCtx(ctx), http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		afterID = page.LastID
	}
}

func (c *Client) ListComplianceOrganizationRoles(ctx context.Context, orgID string) ([]ComplianceRole, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var all []ComplianceRole
	var afterID string
	for {
		q := url.Values{}
		q.Set("limit", "100")
		if afterID != "" {
			q.Set("after_id", afterID)
		}
		var page paginatedComplianceRoles
		path := "/v1/compliance/organizations/" + url.PathEscape(orgID) + "/roles?" + q.Encode()
		if err := c.do(complianceCtx(ctx), http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		afterID = page.LastID
	}
}

func (c *Client) ListComplianceGroups(ctx context.Context) ([]ComplianceGroup, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var all []ComplianceGroup
	var afterID string
	for {
		q := url.Values{}
		q.Set("limit", "100")
		if afterID != "" {
			q.Set("after_id", afterID)
		}
		var page paginatedComplianceGroups
		if err := c.do(complianceCtx(ctx), http.MethodGet, "/v1/compliance/groups?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		afterID = page.LastID
	}
}

func (c *Client) ListComplianceGroupMembers(ctx context.Context, groupID string) ([]ComplianceGroupMember, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var all []ComplianceGroupMember
	var afterID string
	for {
		q := url.Values{}
		q.Set("limit", "100")
		if afterID != "" {
			q.Set("after_id", afterID)
		}
		var page paginatedComplianceGroupMembers
		path := "/v1/compliance/groups/" + url.PathEscape(groupID) + "/members?" + q.Encode()
		if err := c.do(complianceCtx(ctx), http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		afterID = page.LastID
	}
}

func (c *Client) GetComplianceOrganizationSettings(ctx context.Context, orgID string) (*ComplianceOrgSettings, error) {
	if err := c.requireCompliance(); err != nil {
		return nil, err
	}
	var out ComplianceOrgSettings
	path := "/v1/compliance/organizations/" + url.PathEscape(orgID) + "/settings"
	if err := c.do(complianceCtx(ctx), http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func intStr(n int) string {
	// small helper to avoid importing strconv in the many places that need
	// query params.
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 16)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
