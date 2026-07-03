package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ resource.Resource                = &FederationRuleWorkspaceResource{}
	_ resource.ResourceWithImportState = &FederationRuleWorkspaceResource{}
	_ resource.ResourceWithConfigure   = &FederationRuleWorkspaceResource{}
)

func NewFederationRuleWorkspaceResource() resource.Resource {
	return &FederationRuleWorkspaceResource{}
}

type FederationRuleWorkspaceResource struct{ client *anthropic.Client }

type FederationRuleWorkspaceModel struct {
	ID               types.String `tfsdk:"id"`
	FederationRuleID types.String `tfsdk:"federation_rule_id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
}

func (r *FederationRuleWorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_rule_workspace"
}

func (r *FederationRuleWorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Enables a federation rule for an additional workspace. Composite ID `<rule_id>:<workspace_id>`. **Requires OAuth Bearer auth.**",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"federation_rule_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *FederationRuleWorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *FederationRuleWorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FederationRuleWorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.AddFederationRuleWorkspace(ctx, plan.FederationRuleID.ValueString(), plan.WorkspaceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to add federation rule workspace", err.Error())
		return
	}
	plan.ID = types.StringValue(plan.FederationRuleID.ValueString() + ":" + plan.WorkspaceID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FederationRuleWorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FederationRuleWorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.ListFederationRuleWorkspaces(ctx, state.FederationRuleID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read federation rule workspaces", err.Error())
		return
	}
	found := false
	for _, m := range list {
		if m.WorkspaceID == state.WorkspaceID.ValueString() {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *FederationRuleWorkspaceResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes RequireReplace.
}

func (r *FederationRuleWorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FederationRuleWorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveFederationRuleWorkspace(ctx, state.FederationRuleID.ValueString(), state.WorkspaceID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to remove federation rule workspace", err.Error())
	}
}

func (r *FederationRuleWorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected `<rule_id>:<workspace_id>`, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("federation_rule_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
