package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ resource.Resource                = &FederationRuleResource{}
	_ resource.ResourceWithImportState = &FederationRuleResource{}
	_ resource.ResourceWithConfigure   = &FederationRuleResource{}
)

func NewFederationRuleResource() resource.Resource { return &FederationRuleResource{} }

type FederationRuleResource struct{ client *anthropic.Client }

type FederationRuleModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	IssuerID               types.String `tfsdk:"issuer_id"`
	OAuthScope             types.String `tfsdk:"oauth_scope"`
	ServiceAccountID       types.String `tfsdk:"service_account_id"`
	MatchAudience          types.String `tfsdk:"match_audience"`
	MatchSubjectPrefix     types.String `tfsdk:"match_subject_prefix"`
	MatchCondition         types.String `tfsdk:"match_condition"`
	MatchClaims            types.Map    `tfsdk:"match_claims"`
	WorkspaceID            types.String `tfsdk:"workspace_id"`
	AppliesToAllWorkspaces types.Bool   `tfsdk:"applies_to_all_workspaces"`
	TokenLifetimeSeconds   types.Int64  `tfsdk:"token_lifetime_seconds"`
	CreatedAt              types.String `tfsdk:"created_at"`
	ArchivedAt             types.String `tfsdk:"archived_at"`
}

func (r *FederationRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_rule"
}

func (r *FederationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Authorization rule binding an external OIDC identity (via issuer) to a service account. **Requires OAuth Bearer auth.** At least one of `match_subject_prefix` (other than wildcard-only), `match_claims`, or `match_condition` is required. For well-known shared issuers (GitHub Actions, GitLab, Buildkite, Terraform Cloud, Google), tenant identity MUST be constrained via an identity claim or tenant-pinning subject prefix.",
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
			"description": schema.StringAttribute{
				Optional: true, Computed: true,
			},
			"issuer_id": schema.StringAttribute{
				Description:   "Federation issuer ID. Immutable.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"oauth_scope": schema.StringAttribute{
				Description: "Space-separated OAuth scopes (e.g. `workspace:developer workspace:inference`). OAuth callers may only set workspace:developer or workspace:inference; other scopes require Console session.",
				Required:    true,
			},
			"service_account_id": schema.StringAttribute{
				Description: "Service account whose tokens this rule mints.",
				Required:    true,
			},
			"match_audience": schema.StringAttribute{
				Description: "Exact match against the `aud` claim.",
				Optional:    true,
			},
			"match_subject_prefix": schema.StringAttribute{
				Description: "Match the `sub` claim. Exact unless ending with `*` (prefix match).",
				Optional:    true,
			},
			"match_condition": schema.StringAttribute{
				Description: "CEL expression over `claims`. Must evaluate to boolean.",
				Optional:    true,
			},
			"match_claims": schema.MapAttribute{
				Description: "Exact-match `{claim: value}` pairs against top-level string claims.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Initial workspace to enable this rule for. Required unless applies_to_all_workspaces=true. Additional workspaces via `anthropic_federation_rule_workspace`.",
				Optional:    true,
				Computed:    true,
			},
			"applies_to_all_workspaces": schema.BoolAttribute{
				Description: "When true, enables this rule for every workspace (including future ones).",
				Optional:    true,
				Computed:    true,
			},
			"token_lifetime_seconds": schema.Int64Attribute{
				Description: "Lifetime of minted access tokens (60-86400). Defaults to 3600.",
				Optional:    true,
				Computed:    true,
			},
			"created_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *FederationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *FederationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FederationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.CreateFederationRuleRequest{
		IssuerID:    plan.IssuerID.ValueString(),
		Name:        plan.Name.ValueString(),
		OAuthScope:  plan.OAuthScope.ValueString(),
		Description: plan.Description.ValueString(),
		Match:       buildMatch(ctx, plan, &resp.Diagnostics),
		Target: anthropic.FederationTarget{
			Type:             "service_account",
			ServiceAccountID: plan.ServiceAccountID.ValueString(),
		},
		WorkspaceID: plan.WorkspaceID.ValueString(),
	}
	if !plan.AppliesToAllWorkspaces.IsNull() && !plan.AppliesToAllWorkspaces.IsUnknown() {
		v := plan.AppliesToAllWorkspaces.ValueBool()
		in.AppliesToAllWorkspaces = &v
	}
	if !plan.TokenLifetimeSeconds.IsNull() && !plan.TokenLifetimeSeconds.IsUnknown() {
		v := plan.TokenLifetimeSeconds.ValueInt64()
		in.TokenLifetimeSeconds = &v
	}
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.CreateFederationRule(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create federation rule", err.Error())
		return
	}
	federationRuleToModel(ctx, rule, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FederationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FederationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.GetFederationRule(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read federation rule", err.Error())
		return
	}
	federationRuleToModel(ctx, rule, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *FederationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FederationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	match := buildMatch(ctx, plan, &resp.Diagnostics)
	target := anthropic.FederationTarget{Type: "service_account", ServiceAccountID: plan.ServiceAccountID.ValueString()}
	in := anthropic.UpdateFederationRuleRequest{
		Match:  &match,
		Target: &target,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		in.Description = &v
	}
	if !plan.OAuthScope.IsNull() && !plan.OAuthScope.IsUnknown() {
		v := plan.OAuthScope.ValueString()
		in.OAuthScope = &v
	}
	if !plan.AppliesToAllWorkspaces.IsNull() && !plan.AppliesToAllWorkspaces.IsUnknown() {
		v := plan.AppliesToAllWorkspaces.ValueBool()
		in.AppliesToAllWorkspaces = &v
	}
	if !plan.TokenLifetimeSeconds.IsNull() && !plan.TokenLifetimeSeconds.IsUnknown() {
		v := plan.TokenLifetimeSeconds.ValueInt64()
		in.TokenLifetimeSeconds = &v
	}
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.UpdateFederationRule(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update federation rule", err.Error())
		return
	}
	federationRuleToModel(ctx, rule, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FederationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FederationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.ArchiveFederationRule(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to archive federation rule", err.Error())
	}
}

func (r *FederationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildMatch(ctx context.Context, m FederationRuleModel, _ *diag.Diagnostics) anthropic.FederationMatch {
	out := anthropic.FederationMatch{
		Audience:      m.MatchAudience.ValueString(),
		Condition:     m.MatchCondition.ValueString(),
		SubjectPrefix: m.MatchSubjectPrefix.ValueString(),
	}
	if !m.MatchClaims.IsNull() && !m.MatchClaims.IsUnknown() {
		claims := map[string]string{}
		_ = m.MatchClaims.ElementsAs(ctx, &claims, false)
		out.Claims = claims
	}
	return out
}

func federationRuleToModel(ctx context.Context, rule *anthropic.FederationRule, m *FederationRuleModel, _ *diag.Diagnostics) {
	m.ID = types.StringValue(rule.ID)
	m.Name = types.StringValue(rule.Name)
	m.Description = stringValueOrEmpty(rule.Description)
	m.IssuerID = types.StringValue(rule.IssuerID)
	m.OAuthScope = types.StringValue(rule.OAuthScope)
	m.ServiceAccountID = types.StringValue(rule.Target.ServiceAccountID)
	m.MatchAudience = stringValueOrEmpty(rule.Match.Audience)
	m.MatchSubjectPrefix = stringValueOrEmpty(rule.Match.SubjectPrefix)
	m.MatchCondition = stringValueOrEmpty(rule.Match.Condition)
	if rule.Match.Claims == nil {
		m.MatchClaims = types.MapNull(types.StringType)
	} else {
		v, _ := types.MapValueFrom(ctx, types.StringType, rule.Match.Claims)
		m.MatchClaims = v
	}
	m.WorkspaceID = stringValueOrEmpty(rule.WorkspaceID)
	m.AppliesToAllWorkspaces = types.BoolValue(rule.AppliesToAllWorkspaces)
	m.TokenLifetimeSeconds = types.Int64Value(rule.TokenLifetimeSeconds)
	m.CreatedAt = types.StringValue(rule.CreatedAt)
	m.ArchivedAt = stringValueOrEmpty(rule.ArchivedAt)
}
