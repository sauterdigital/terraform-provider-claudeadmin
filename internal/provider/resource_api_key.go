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
	_ resource.Resource                = &APIKeyResource{}
	_ resource.ResourceWithImportState = &APIKeyResource{}
	_ resource.ResourceWithConfigure   = &APIKeyResource{}
)

func NewAPIKeyResource() resource.Resource { return &APIKeyResource{} }

type APIKeyResource struct{ client *anthropic.Client }

type APIKeyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	PartialKeyHint types.String `tfsdk:"partial_key_hint"`
	WorkspaceID    types.String `tfsdk:"workspace_id"`
	CreatedByID    types.String `tfsdk:"created_by_id"`
	CreatedByType  types.String `tfsdk:"created_by_type"`
}

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an existing Anthropic API key. The Admin API does NOT support creating keys — keys must be created in the Console first, then their `id` supplied here so this provider can manage `name` and `status` declaratively. Destroying this resource sets status to `archived`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "ID of an existing API key (e.g. `apikey_...`). Must already exist in your organization.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable key name.",
				Optional:    true,
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Key status: `active`, `inactive`, or `archived`. `expired` is read-only (set by the API).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("active", "inactive", "archived"),
				},
			},
			"created_at":       schema.StringAttribute{Computed: true},
			"expires_at":       schema.StringAttribute{Computed: true},
			"partial_key_hint": schema.StringAttribute{Computed: true},
			"workspace_id":     schema.StringAttribute{Computed: true},
			"created_by_id":    schema.StringAttribute{Computed: true},
			"created_by_type":  schema.StringAttribute{Computed: true},
		},
	}
}

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetAPIKey(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API key not found", "The Admin API cannot create keys — the supplied id must already exist. "+err.Error())
		return
	}

	in := apiKeyUpdateFromPlan(plan, existing)
	updated, err := r.client.UpdateAPIKey(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update API key during create", err.Error())
		return
	}
	apiKeyToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := r.client.GetAPIKey(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read API key", err.Error())
		return
	}
	apiKeyToModel(k, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := apiKeyUpdateFromPlan(plan, nil)
	k, err := r.client.UpdateAPIKey(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update API key", err.Error())
		return
	}
	apiKeyToModel(k, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	archived := "archived"
	if _, err := r.client.UpdateAPIKey(ctx, state.ID.ValueString(), anthropic.UpdateAPIKeyRequest{Status: &archived}); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to archive API key", err.Error())
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func apiKeyUpdateFromPlan(plan APIKeyResourceModel, _ *anthropic.APIKey) anthropic.UpdateAPIKeyRequest {
	var in anthropic.UpdateAPIKeyRequest
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		in.Name = &v
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		v := plan.Status.ValueString()
		in.Status = &v
	}
	return in
}

func apiKeyToModel(k *anthropic.APIKey, m *APIKeyResourceModel) {
	m.ID = types.StringValue(k.ID)
	m.Name = types.StringValue(k.Name)
	m.Status = types.StringValue(k.Status)
	m.CreatedAt = types.StringValue(k.CreatedAt)
	m.ExpiresAt = optionalStringValue(k.ExpiresAt)
	m.PartialKeyHint = types.StringValue(k.PartialKeyHint)
	m.WorkspaceID = optionalStringValue(k.WorkspaceID)
	m.CreatedByID = types.StringValue(k.CreatedBy.ID)
	m.CreatedByType = types.StringValue(k.CreatedBy.Type)
}
