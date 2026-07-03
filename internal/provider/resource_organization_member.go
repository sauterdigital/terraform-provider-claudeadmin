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
	_ resource.Resource                = &OrganizationMemberResource{}
	_ resource.ResourceWithImportState = &OrganizationMemberResource{}
	_ resource.ResourceWithConfigure   = &OrganizationMemberResource{}
)

func NewOrganizationMemberResource() resource.Resource { return &OrganizationMemberResource{} }

type OrganizationMemberResource struct{ client *anthropic.Client }

type OrganizationMemberResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Email   types.String `tfsdk:"email"`
	Name    types.String `tfsdk:"name"`
	Role    types.String `tfsdk:"role"`
	AddedAt types.String `tfsdk:"added_at"`
}

func (r *OrganizationMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member"
}

func (r *OrganizationMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the organization-level role of an existing user. Users must join via an accepted `anthropic_invite` first; this resource lets you change their role declaratively or remove them. Destroy removes the user from the organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "User ID (e.g. `user_...`). Must already exist as an organization member.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				Description: "Organization role: user, developer, billing, or claude_code_user. `admin` is not settable via this API.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("user", "developer", "billing", "claude_code_user"),
				},
			},
			"email":    schema.StringAttribute{Computed: true},
			"name":     schema.StringAttribute{Computed: true},
			"added_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *OrganizationMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *OrganizationMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.GetUser(ctx, plan.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("User not found", "Users can only be added via `anthropic_invite`. The supplied id must already exist. "+err.Error())
		return
	}
	u, err := r.client.UpdateUser(ctx, plan.ID.ValueString(), anthropic.UpdateUserRequest{Role: plan.Role.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set user role", err.Error())
		return
	}
	userToModel(u, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *OrganizationMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := r.client.GetUser(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read user", err.Error())
		return
	}
	userToModel(u, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *OrganizationMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := r.client.UpdateUser(ctx, plan.ID.ValueString(), anthropic.UpdateUserRequest{Role: plan.Role.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update user role", err.Error())
		return
	}
	userToModel(u, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *OrganizationMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUser(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to remove organization member", err.Error())
	}
}

func (r *OrganizationMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func userToModel(u *anthropic.User, m *OrganizationMemberResourceModel) {
	m.ID = types.StringValue(u.ID)
	m.Email = types.StringValue(u.Email)
	m.Name = types.StringValue(u.Name)
	m.Role = types.StringValue(u.Role)
	m.AddedAt = types.StringValue(u.AddedAt)
}
