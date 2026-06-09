package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var (
	_ resource.Resource                = &WorkspaceResource{}
	_ resource.ResourceWithImportState = &WorkspaceResource{}
	_ resource.ResourceWithConfigure   = &WorkspaceResource{}
)

func NewWorkspaceResource() resource.Resource { return &WorkspaceResource{} }

type WorkspaceResource struct {
	client *anthropic.Client
}

type WorkspaceResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Tags          types.Map    `tfsdk:"tags"`
	CreatedAt     types.String `tfsdk:"created_at"`
	ArchivedAt    types.String `tfsdk:"archived_at"`
	DisplayColor  types.String `tfsdk:"display_color"`
	CompartmentID types.String `tfsdk:"compartment_id"`
	ExternalKeyID types.String `tfsdk:"external_key_id"`
	DataResidency types.Object `tfsdk:"data_residency"`
}

func (r *WorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (r *WorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic organization workspace. Deletion archives the workspace; the underlying API does not support hard deletion.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Workspace ID (e.g. `wrkspc_...`).",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable workspace name.",
				Required:    true,
			},
			"tags": schema.MapAttribute{
				Description: "User-defined tags as string key-value pairs. Keys may not begin with `anthropic`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"external_key_id": schema.StringAttribute{
				Description:   "ID of the customer-managed encryption key (CMEK) configuration. Write-once: cannot be changed after creation.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Description:   "RFC 3339 timestamp the workspace was created.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"archived_at": schema.StringAttribute{
				Description: "RFC 3339 timestamp the workspace was archived, null while active.",
				Computed:    true,
			},
			"display_color": schema.StringAttribute{
				Description: "Hex display color assigned by the platform.",
				Computed:    true,
			},
			"compartment_id": schema.StringAttribute{
				Description:   "Identifier for this workspace's CMEK encryption compartment.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"data_residency": schema.SingleNestedAttribute{
				Description: "Data residency configuration. Any change forces resource replacement because the API marks `workspace_geo` immutable after creation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"workspace_geo": schema.StringAttribute{
						Description:   "Geographic region for workspace data storage. Immutable after creation. Defaults to `us` when omitted.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"default_inference_geo": schema.StringAttribute{
						Description:   "Default inference geo applied when requests omit the parameter. Defaults to `global`.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"allowed_inference_geos": schema.ListAttribute{
						Description:   "Permitted inference geo values. Pass `[\"unrestricted\"]` to allow all geos (the provider serializes this single-element list as the string `\"unrestricted\"` for the API).",
						Optional:      true,
						Computed:      true,
						ElementType:   types.StringType,
						PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
					},
				},
			},
		},
	}
}

func (r *WorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *WorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkspaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := tagsFromTerraform(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := anthropic.CreateWorkspaceRequest{
		Name: plan.Name.ValueString(),
		Tags: tags,
	}
	if !plan.ExternalKeyID.IsNull() && !plan.ExternalKeyID.IsUnknown() {
		in.ExternalKeyID = plan.ExternalKeyID.ValueString()
	}
	dr, drDiags := dataResidencyFromTerraform(ctx, plan.DataResidency)
	resp.Diagnostics.Append(drDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	in.DataResidency = dr

	ws, err := r.client.CreateWorkspace(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create workspace", err.Error())
		return
	}
	resp.Diagnostics.Append(workspaceToModel(ctx, ws, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *WorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkspaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ws, err := r.client.GetWorkspace(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read workspace", err.Error())
		return
	}
	resp.Diagnostics.Append(workspaceToModel(ctx, ws, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *WorkspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkspaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := tagsFromTerraform(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ws, err := r.client.UpdateWorkspace(ctx, plan.ID.ValueString(), anthropic.UpdateWorkspaceRequest{
		Name: plan.Name.ValueString(),
		Tags: tags,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update workspace", err.Error())
		return
	}
	resp.Diagnostics.Append(workspaceToModel(ctx, ws, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *WorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkspaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.ArchiveWorkspace(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to archive workspace", err.Error())
	}
}

func (r *WorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func workspaceToModel(ctx context.Context, ws *anthropic.Workspace, m *WorkspaceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = types.StringValue(ws.ID)
	m.Name = types.StringValue(ws.Name)
	m.CreatedAt = types.StringValue(ws.CreatedAt)
	if ws.ArchivedAt != nil {
		m.ArchivedAt = types.StringValue(*ws.ArchivedAt)
	} else {
		m.ArchivedAt = types.StringNull()
	}
	m.DisplayColor = types.StringValue(ws.DisplayColor)
	m.CompartmentID = stringValueOrEmpty(ws.CompartmentID)
	m.ExternalKeyID = stringValueOrEmpty(ws.ExternalKeyID)

	tags, tagDiags := tagsMapValue(ctx, ws.Tags)
	diags.Append(tagDiags...)
	m.Tags = tags

	dr, drDiags := dataResidencyObjectValue(ctx, ws.DataResidency)
	diags.Append(drDiags...)
	m.DataResidency = dr
	return diags
}
