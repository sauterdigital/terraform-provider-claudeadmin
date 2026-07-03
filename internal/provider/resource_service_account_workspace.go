package provider

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &ServiceAccountWorkspaceResource{}
	_ resource.ResourceWithImportState = &ServiceAccountWorkspaceResource{}
	_ resource.ResourceWithConfigure   = &ServiceAccountWorkspaceResource{}
)

func NewServiceAccountWorkspaceResource() resource.Resource {
	return &ServiceAccountWorkspaceResource{}
}

type ServiceAccountWorkspaceResource struct{ client *anthropic.Client }

type ServiceAccountWorkspaceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	WorkspaceRole    types.String `tfsdk:"workspace_role"`
	Implicit         types.Bool   `tfsdk:"implicit"`
}

func (r *ServiceAccountWorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_workspace"
}

func (r *ServiceAccountWorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a service account to a workspace with a given role. **Requires OAuth Bearer auth**. Composite ID `<service_account_id>:<workspace_id>` (used for import). Every SA is an implicit `workspace_user` of the default workspace; adding it explicitly assigns a chosen role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_account_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workspace_role": schema.StringAttribute{
				Description: "Role: workspace_user, workspace_developer, workspace_restricted_developer, or workspace_admin. Service accounts cannot hold workspace_billing.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("workspace_user", "workspace_developer", "workspace_restricted_developer", "workspace_admin"),
				},
			},
			"implicit": schema.BoolAttribute{
				Description: "True only for the implicit default-workspace membership every SA has. Implicit memberships cannot be removed.",
				Computed:    true,
			},
		},
	}
}

func (r *ServiceAccountWorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *ServiceAccountWorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceAccountWorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.AddWorkspaceToServiceAccount(ctx, plan.ServiceAccountID.ValueString(), anthropic.AddWorkspaceToServiceAccountRequest{
		WorkspaceID:   plan.WorkspaceID.ValueString(),
		WorkspaceRole: plan.WorkspaceRole.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to add SA to workspace", err.Error())
		return
	}
	saWorkspaceToModel(m, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ServiceAccountWorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceAccountWorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.GetServiceAccountWorkspaceMember(ctx, state.WorkspaceID.ValueString(), state.ServiceAccountID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SA workspace membership", err.Error())
		return
	}
	saWorkspaceToModel(m, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ServiceAccountWorkspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceAccountWorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.UpdateServiceAccountWorkspaceRole(ctx, plan.WorkspaceID.ValueString(), plan.ServiceAccountID.ValueString(), plan.WorkspaceRole.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SA workspace role", err.Error())
		return
	}
	saWorkspaceToModel(m, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ServiceAccountWorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceAccountWorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveWorkspaceFromServiceAccount(ctx, state.ServiceAccountID.ValueString(), state.WorkspaceID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to remove SA from workspace", err.Error())
	}
}

func (r *ServiceAccountWorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected `<service_account_id>:<workspace_id>`, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func saWorkspaceToModel(m *anthropic.ServiceAccountWorkspaceMember, out *ServiceAccountWorkspaceModel) {
	out.ID = types.StringValue(m.ServiceAccountID + ":" + m.WorkspaceID)
	out.ServiceAccountID = types.StringValue(m.ServiceAccountID)
	out.WorkspaceID = types.StringValue(m.WorkspaceID)
	out.WorkspaceRole = types.StringValue(m.WorkspaceRole)
	out.Implicit = types.BoolValue(m.Implicit)
}
