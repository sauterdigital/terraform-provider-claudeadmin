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
	_ resource.Resource                = &WorkspaceMemberResource{}
	_ resource.ResourceWithImportState = &WorkspaceMemberResource{}
	_ resource.ResourceWithConfigure   = &WorkspaceMemberResource{}
)

func NewWorkspaceMemberResource() resource.Resource { return &WorkspaceMemberResource{} }

type WorkspaceMemberResource struct{ client *anthropic.Client }

type WorkspaceMemberResourceModel struct {
	ID            types.String `tfsdk:"id"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	UserID        types.String `tfsdk:"user_id"`
	WorkspaceRole types.String `tfsdk:"workspace_role"`
}

func (r *WorkspaceMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_member"
}

func (r *WorkspaceMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Adds an existing organization user to a workspace with a given role. The composite ID is `<workspace_id>:<user_id>` (used for import).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Composite ID `<workspace_id>:<user_id>`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": schema.StringAttribute{
				Description:   "Target workspace ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user_id": schema.StringAttribute{
				Description:   "User ID to add. The user must already be an organization member.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workspace_role": schema.StringAttribute{
				Description: "Role: workspace_user, workspace_developer, workspace_restricted_developer, workspace_admin, or workspace_billing (workspace_billing cannot be set at creation — only via update).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("workspace_user", "workspace_developer", "workspace_restricted_developer", "workspace_admin", "workspace_billing"),
				},
			},
		},
	}
}

func (r *WorkspaceMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *WorkspaceMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkspaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m, err := r.client.AddWorkspaceMember(ctx, plan.WorkspaceID.ValueString(), anthropic.CreateWorkspaceMemberRequest{
		UserID:        plan.UserID.ValueString(),
		WorkspaceRole: plan.WorkspaceRole.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to add workspace member", err.Error())
		return
	}
	workspaceMemberToModel(m, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *WorkspaceMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkspaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.GetWorkspaceMember(ctx, state.WorkspaceID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read workspace member", err.Error())
		return
	}
	workspaceMemberToModel(m, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *WorkspaceMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkspaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.UpdateWorkspaceMember(ctx, plan.WorkspaceID.ValueString(), plan.UserID.ValueString(), anthropic.UpdateWorkspaceMemberRequest{
		WorkspaceRole: plan.WorkspaceRole.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update workspace member", err.Error())
		return
	}
	workspaceMemberToModel(m, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *WorkspaceMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkspaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteWorkspaceMember(ctx, state.WorkspaceID.ValueString(), state.UserID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to remove workspace member", err.Error())
	}
}

func (r *WorkspaceMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected `<workspace_id>:<user_id>`, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func workspaceMemberToModel(m *anthropic.WorkspaceMember, out *WorkspaceMemberResourceModel) {
	out.ID = types.StringValue(m.WorkspaceID + ":" + m.UserID)
	out.WorkspaceID = types.StringValue(m.WorkspaceID)
	out.UserID = types.StringValue(m.UserID)
	out.WorkspaceRole = types.StringValue(m.WorkspaceRole)
}
