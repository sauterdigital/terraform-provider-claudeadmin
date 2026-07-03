package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ resource.Resource                = &FederationIssuerResource{}
	_ resource.ResourceWithImportState = &FederationIssuerResource{}
	_ resource.ResourceWithConfigure   = &FederationIssuerResource{}
)

func NewFederationIssuerResource() resource.Resource { return &FederationIssuerResource{} }

type FederationIssuerResource struct{ client *anthropic.Client }

type FederationIssuerModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	IssuerURL             types.String `tfsdk:"issuer_url"`
	CheckJTI              types.Bool   `tfsdk:"check_jti"`
	MaxJWTLifetimeSeconds types.Int64  `tfsdk:"max_jwt_lifetime_seconds"`
	JWKSType              types.String `tfsdk:"jwks_type"`
	JWKSDiscoveryBase     types.String `tfsdk:"jwks_discovery_base"`
	JWKSURL               types.String `tfsdk:"jwks_url"`
	JWKSCACertPEM         types.String `tfsdk:"jwks_ca_cert_pem"`
	JWKSKeysJSON          types.String `tfsdk:"jwks_keys_json"`
	CreatedAt             types.String `tfsdk:"created_at"`
	ArchivedAt            types.String `tfsdk:"archived_at"`
}

func (r *FederationIssuerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_issuer"
}

func (r *FederationIssuerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers an external OIDC issuer for RFC 7523 jwt-bearer federation (GitHub Actions, GitLab, Buildkite, Terraform Cloud, Google, etc). **Requires OAuth Bearer auth.** The JWKS source is polymorphic — pick `discovery` (default), `explicit_url`, or `inline` and fill the matching fields.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:   "Slug identifier. Unique within the org. Immutable after creation.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"issuer_url": schema.StringAttribute{
				Description: "The `iss` claim value to match against. JWTs must match exactly.",
				Required:    true,
			},
			"check_jti": schema.BoolAttribute{
				Description: "Enforce JTI single-use (replay protection). Defaults to true.",
				Optional:    true,
				Computed:    true,
			},
			"max_jwt_lifetime_seconds": schema.Int64Attribute{
				Description: "Max iat→exp spread (1-176400 seconds). Defaults to 3600.",
				Optional:    true,
				Computed:    true,
			},
			"jwks_type": schema.StringAttribute{
				Description: "How signing keys are obtained: `discovery` (default, fetched via OIDC discovery), `explicit_url` (fixed JWKS endpoint), or `inline` (static keys).",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("discovery", "explicit_url", "inline")},
			},
			"jwks_discovery_base": schema.StringAttribute{
				Description: "discovery mode only — override the discovery base URL when it differs from issuer_url.",
				Optional:    true,
			},
			"jwks_url": schema.StringAttribute{
				Description: "explicit_url mode only — the JWKS endpoint.",
				Optional:    true,
			},
			"jwks_ca_cert_pem": schema.StringAttribute{
				Description: "Optional custom CA (PEM) for TLS verification of the JWKS fetch.",
				Optional:    true,
			},
			"jwks_keys_json": schema.StringAttribute{
				Description: "inline mode only — JSON array of JWK objects.",
				Optional:    true,
			},
			"created_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *FederationIssuerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *FederationIssuerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FederationIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.CreateFederationIssuerRequest{
		IssuerURL: plan.IssuerURL.ValueString(),
		Name:      plan.Name.ValueString(),
	}
	if !plan.CheckJTI.IsNull() && !plan.CheckJTI.IsUnknown() {
		v := plan.CheckJTI.ValueBool()
		in.CheckJTI = &v
	}
	if !plan.MaxJWTLifetimeSeconds.IsNull() && !plan.MaxJWTLifetimeSeconds.IsUnknown() {
		v := plan.MaxJWTLifetimeSeconds.ValueInt64()
		in.MaxJWTLifetimeSeconds = &v
	}
	if jwks, ok := buildJWKSFromModel(&plan, &resp.Diagnostics); ok && jwks != nil {
		in.JWKS = jwks
	}
	if resp.Diagnostics.HasError() {
		return
	}

	iss, err := r.client.CreateFederationIssuer(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create federation issuer", err.Error())
		return
	}
	federationIssuerToModel(iss, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FederationIssuerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FederationIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	iss, err := r.client.GetFederationIssuer(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read federation issuer", err.Error())
		return
	}
	federationIssuerToModel(iss, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *FederationIssuerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FederationIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.UpdateFederationIssuerRequest{}
	if !plan.IssuerURL.IsNull() && !plan.IssuerURL.IsUnknown() {
		v := plan.IssuerURL.ValueString()
		in.IssuerURL = &v
	}
	if !plan.CheckJTI.IsNull() && !plan.CheckJTI.IsUnknown() {
		v := plan.CheckJTI.ValueBool()
		in.CheckJTI = &v
	}
	if !plan.MaxJWTLifetimeSeconds.IsNull() && !plan.MaxJWTLifetimeSeconds.IsUnknown() {
		v := plan.MaxJWTLifetimeSeconds.ValueInt64()
		in.MaxJWTLifetimeSeconds = &v
	}
	if jwks, ok := buildJWKSFromModel(&plan, &resp.Diagnostics); ok && jwks != nil {
		in.JWKS = jwks
	}
	if resp.Diagnostics.HasError() {
		return
	}

	iss, err := r.client.UpdateFederationIssuer(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update federation issuer", err.Error())
		return
	}
	federationIssuerToModel(iss, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FederationIssuerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FederationIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.ArchiveFederationIssuer(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to archive federation issuer", err.Error())
	}
}

func (r *FederationIssuerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildJWKSFromModel constructs the polymorphic JWKS payload from the flat
// schema fields. Returns (nil, true) if the user didn't set jwks_type (let the
// API default to discovery). Adds a diagnostic on inline keys parse failure.
func buildJWKSFromModel(m *FederationIssuerModel, diags *diag.Diagnostics) (*anthropic.IssuerJWKS, bool) {
	if m.JWKSType.IsNull() || m.JWKSType.IsUnknown() {
		return nil, true
	}
	out := &anthropic.IssuerJWKS{Type: m.JWKSType.ValueString()}
	if !m.JWKSCACertPEM.IsNull() {
		out.CACertPEM = m.JWKSCACertPEM.ValueString()
	}
	switch out.Type {
	case "discovery":
		if !m.JWKSDiscoveryBase.IsNull() {
			out.DiscoveryBase = m.JWKSDiscoveryBase.ValueString()
		}
	case "explicit_url":
		if !m.JWKSURL.IsNull() {
			out.URL = m.JWKSURL.ValueString()
		}
	case "inline":
		if m.JWKSKeysJSON.IsNull() || m.JWKSKeysJSON.ValueString() == "" {
			diags.AddError("jwks_keys_json required", "jwks_type=inline requires a JSON array of JWK objects in jwks_keys_json.")
			return nil, false
		}
		var keys []json.RawMessage
		if err := json.Unmarshal([]byte(m.JWKSKeysJSON.ValueString()), &keys); err != nil {
			diags.AddError("Invalid jwks_keys_json", err.Error())
			return nil, false
		}
		out.Keys = keys
	}
	return out, true
}

func federationIssuerToModel(iss *anthropic.FederationIssuer, m *FederationIssuerModel) {
	m.ID = types.StringValue(iss.ID)
	m.Name = types.StringValue(iss.Name)
	m.IssuerURL = types.StringValue(iss.IssuerURL)
	m.CheckJTI = types.BoolValue(iss.CheckJTI)
	m.MaxJWTLifetimeSeconds = types.Int64Value(iss.MaxJWTLifetimeSeconds)
	m.JWKSType = types.StringValue(iss.JWKS.Type)
	m.JWKSDiscoveryBase = stringValueOrEmpty(iss.JWKS.DiscoveryBase)
	m.JWKSURL = stringValueOrEmpty(iss.JWKS.URL)
	m.JWKSCACertPEM = stringValueOrEmpty(iss.JWKS.CACertPEM)
	if iss.JWKS.Keys != nil {
		raw, _ := json.Marshal(iss.JWKS.Keys)
		m.JWKSKeysJSON = types.StringValue(string(raw))
	} else {
		m.JWKSKeysJSON = types.StringNull()
	}
	m.CreatedAt = types.StringValue(iss.CreatedAt)
	m.ArchivedAt = stringValueOrEmpty(iss.ArchivedAt)
}
