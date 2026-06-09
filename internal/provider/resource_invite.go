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

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ resource.Resource                = &InviteResource{}
	_ resource.ResourceWithImportState = &InviteResource{}
	_ resource.ResourceWithConfigure   = &InviteResource{}
)

func NewInviteResource() resource.Resource { return &InviteResource{} }

type InviteResource struct{ client *anthropic.Client }

type InviteResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	Role      types.String `tfsdk:"role"`
	Status    types.String `tfsdk:"status"`
	InvitedAt types.String `tfsdk:"invited_at"`
	ExpiresAt types.String `tfsdk:"expires_at"`
}

func (r *InviteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_invite"
}

func (r *InviteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates an Anthropic organization invite. Invites are immutable — changes to email or role force replacement. The `admin` role cannot be granted via invite.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"email": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				Description:   "Org role: user, developer, billing, or claude_code_user. `admin` is not allowed.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("user", "developer", "billing", "claude_code_user"),
				},
			},
			"status":     schema.StringAttribute{Computed: true},
			"invited_at": schema.StringAttribute{Computed: true},
			"expires_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *InviteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *InviteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InviteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inv, err := r.client.CreateInvite(ctx, anthropic.CreateInviteRequest{
		Email: plan.Email.ValueString(),
		Role:  plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create invite", err.Error())
		return
	}
	inviteToModel(inv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *InviteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InviteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inv, err := r.client.GetInvite(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read invite", err.Error())
		return
	}
	inviteToModel(inv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *InviteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes RequireReplace; this method is unreachable in practice.
	var plan InviteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *InviteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InviteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInvite(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete invite", err.Error())
	}
}

func (r *InviteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func inviteToModel(inv *anthropic.Invite, m *InviteResourceModel) {
	m.ID = types.StringValue(inv.ID)
	m.Email = types.StringValue(inv.Email)
	m.Role = types.StringValue(inv.Role)
	m.Status = types.StringValue(inv.Status)
	m.InvitedAt = types.StringValue(inv.InvitedAt)
	m.ExpiresAt = types.StringValue(inv.ExpiresAt)
}
