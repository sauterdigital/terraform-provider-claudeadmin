package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &ServiceAccountResource{}
	_ resource.ResourceWithImportState = &ServiceAccountResource{}
	_ resource.ResourceWithConfigure   = &ServiceAccountResource{}
)

func NewServiceAccountResource() resource.Resource { return &ServiceAccountResource{} }

type ServiceAccountResource struct{ client *anthropic.Client }

type ServiceAccountModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	OrganizationRole types.String `tfsdk:"organization_role"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	ArchivedAt       types.String `tfsdk:"archived_at"`
}

func (r *ServiceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *ServiceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service account — a named non-human identity that federation rules target. **Requires OAuth Bearer auth** (`oauth_token` / `ANTHROPIC_OAUTH_TOKEN`); Admin API keys are rejected. Creating an `admin`-role service account additionally requires an interactive credential (user OAuth or Console session); a workload-issued token may only create `developer`-role accounts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:   "Slug identifier (lowercase, digits, hyphens). Unique within the org; duplicate returns 409. Immutable after creation.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"organization_role": schema.StringAttribute{
				Description: "Org-level role: `developer` (default) or `admin`. A federation rule may only grant org:admin scope when this is admin.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("developer", "admin")},
			},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *ServiceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *ServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.CreateServiceAccountRequest{
		Name:             plan.Name.ValueString(),
		Description:      plan.Description.ValueString(),
		OrganizationRole: plan.OrganizationRole.ValueString(),
	}
	sa, err := r.client.CreateServiceAccount(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create service account", err.Error())
		return
	}
	serviceAccountToModel(sa, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sa, err := r.client.GetServiceAccount(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read service account", err.Error())
		return
	}
	serviceAccountToModel(sa, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.UpdateServiceAccountRequest{}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		in.Description = &v
	}
	if !plan.OrganizationRole.IsNull() && !plan.OrganizationRole.IsUnknown() {
		v := plan.OrganizationRole.ValueString()
		in.OrganizationRole = &v
	}
	sa, err := r.client.UpdateServiceAccount(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update service account", err.Error())
		return
	}
	serviceAccountToModel(sa, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.ArchiveServiceAccount(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to archive service account", err.Error())
	}
}

func (r *ServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func serviceAccountToModel(sa *anthropic.ServiceAccount, m *ServiceAccountModel) {
	m.ID = types.StringValue(sa.ID)
	m.Name = types.StringValue(sa.Name)
	m.Description = types.StringValue(sa.Description)
	m.OrganizationRole = types.StringValue(sa.OrganizationRole)
	m.CreatedAt = types.StringValue(sa.CreatedAt)
	m.UpdatedAt = types.StringValue(sa.UpdatedAt)
	m.ArchivedAt = stringValueOrEmpty(sa.ArchivedAt)
}
