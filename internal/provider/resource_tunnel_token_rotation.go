package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

var (
	_ resource.Resource                = &TunnelTokenRotationResource{}
	_ resource.ResourceWithConfigure   = &TunnelTokenRotationResource{}
	_ resource.ResourceWithImportState = &TunnelTokenRotationResource{}
)

func NewTunnelTokenRotationResource() resource.Resource {
	return &TunnelTokenRotationResource{}
}

type TunnelTokenRotationResource struct{ client *anthropic.Client }

type TunnelTokenRotationModel struct {
	ID          types.String `tfsdk:"id"`
	TunnelID    types.String `tfsdk:"tunnel_id"`
	RotationID  types.String `tfsdk:"rotation_id"`
	Reason      types.String `tfsdk:"reason"`
	TunnelToken types.String `tfsdk:"tunnel_token"`
}

func (r *TunnelTokenRotationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunnel_token_rotation"
}

func (r *TunnelTokenRotationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Rotates the connection token for an MCP tunnel. Rotation is a one-way, irreversible operation — changing `rotation_id` forces replacement, which triggers a new `POST /v1/tunnels/{id}/rotate_token` call and produces a fresh `tunnel_token`. Beta — requires OAuth Bearer auth + beta header `anthropic-beta: mcp-tunnels-2026-06-22` (added automatically). To simply read the current token without rotating, use the `anthropic_tunnel_token` data source instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Composite identifier `<tunnel_id>:<rotation_id>` — deterministic per rotation.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tunnel_id": schema.StringAttribute{
				Description:   "The tunnel whose token should be rotated.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"rotation_id": schema.StringAttribute{
				Description:   "User-chosen nonce/label for this rotation (e.g. `2026-Q3`, or a timestamp). Change this value to trigger a new rotation.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"reason": schema.StringAttribute{
				Description:   "Optional free-form reason recorded in the audit log (e.g. `quarterly rotation`, `key exposure`).",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tunnel_token": schema.StringAttribute{
				Description: "The freshly-rotated tunnel token. **Sensitive** — write to a Vault / K8s Secret via output. Not refreshed on subsequent reads (rotation is one-shot).",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *TunnelTokenRotationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *TunnelTokenRotationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TunnelTokenRotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.RotateTunnelToken(ctx, plan.TunnelID.ValueString(), plan.Reason.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to rotate tunnel token", err.Error())
		return
	}
	plan.ID = types.StringValue(plan.TunnelID.ValueString() + ":" + plan.RotationID.ValueString())
	plan.TunnelToken = types.StringValue(result.TunnelToken)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TunnelTokenRotationResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Tunnel tokens have no read endpoint that reflects the last rotation nonce.
	// State is authoritative until rotation_id changes.
}

func (r *TunnelTokenRotationResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All mutable attributes RequiresReplace; Update is unreachable.
}

func (r *TunnelTokenRotationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Rotation is not reversible. Removing from state does not affect the tunnel.
}

func (r *TunnelTokenRotationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
