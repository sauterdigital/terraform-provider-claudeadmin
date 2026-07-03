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
	_ resource.Resource                = &TunnelCertificateResource{}
	_ resource.ResourceWithImportState = &TunnelCertificateResource{}
	_ resource.ResourceWithConfigure   = &TunnelCertificateResource{}
)

func NewTunnelCertificateResource() resource.Resource { return &TunnelCertificateResource{} }

type TunnelCertificateResource struct{ client *anthropic.Client }

type TunnelCertificateModel struct {
	ID               types.String `tfsdk:"id"`
	TunnelID         types.String `tfsdk:"tunnel_id"`
	CACertificatePEM types.String `tfsdk:"ca_certificate_pem"`
	Fingerprint      types.String `tfsdk:"fingerprint"`
	CreatedAt        types.String `tfsdk:"created_at"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	ArchivedAt       types.String `tfsdk:"archived_at"`
}

func (r *TunnelCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunnel_certificate"
}

func (r *TunnelCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers a public CA certificate for an MCP tunnel. **Requires OAuth Bearer auth + beta header `anthropic-beta: mcp-tunnels-2026-06-22` (added automatically).** PEM must contain exactly one X.509 cert with no private-key material. A tunnel holds at most two non-archived certificates. Composite import id: `<tunnel_id>:<cert_id>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tunnel_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ca_certificate_pem": schema.StringAttribute{
				Description:   "PEM-encoded X.509 CA certificate. Immutable after creation.",
				Required:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"fingerprint": schema.StringAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"expires_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *TunnelCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *TunnelCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TunnelCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cert, err := r.client.CreateTunnelCertificate(ctx, plan.TunnelID.ValueString(), plan.CACertificatePEM.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create tunnel certificate", err.Error())
		return
	}
	tunnelCertToModel(cert, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TunnelCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TunnelCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cert, err := r.client.GetTunnelCertificate(ctx, state.TunnelID.ValueString(), state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read tunnel certificate", err.Error())
		return
	}
	tunnelCertToModel(cert, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *TunnelCertificateResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All mutable attributes are RequireReplace.
}

func (r *TunnelCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TunnelCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.ArchiveTunnelCertificate(ctx, state.TunnelID.ValueString(), state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to archive tunnel certificate", err.Error())
	}
}

func (r *TunnelCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected `<tunnel_id>:<cert_id>`, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tunnel_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func tunnelCertToModel(c *anthropic.TunnelCertificate, m *TunnelCertificateModel) {
	m.ID = types.StringValue(c.ID)
	m.TunnelID = types.StringValue(c.TunnelID)
	m.Fingerprint = types.StringValue(c.Fingerprint)
	m.CreatedAt = types.StringValue(c.CreatedAt)
	m.ExpiresAt = optionalStringValue(c.ExpiresAt)
	m.ArchivedAt = optionalStringValue(c.ArchivedAt)
}
