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
	_ resource.Resource                = &ExternalKeyResource{}
	_ resource.ResourceWithImportState = &ExternalKeyResource{}
	_ resource.ResourceWithConfigure   = &ExternalKeyResource{}
)

func NewExternalKeyResource() resource.Resource { return &ExternalKeyResource{} }

type ExternalKeyResource struct{ client *anthropic.Client }

func (r *ExternalKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_key"
}

func (r *ExternalKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a customer-managed encryption key (CMEK) configuration for use with workspaces. Requires CMEK to be enabled for your organization. `provider_config.geo` and `provider_config` itself become immutable once any workspace references this key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"display_name": schema.StringAttribute{Required: true},
			"geo": schema.StringAttribute{
				Description: "Currently only `us` is supported.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("us"),
				},
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
			"provider_config": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Provider: aws, gcp, or azure.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "gcp", "azure"),
						},
					},
					// AWS
					"kms_arn":  schema.StringAttribute{Optional: true, Description: "AWS only — the KMS key ARN."},
					"role_arn": schema.StringAttribute{Optional: true, Description: "AWS only — the IAM role ARN Anthropic should assume."},
					"region":   schema.StringAttribute{Optional: true, Description: "AWS only — KMS region."},
					// GCP + Azure share key_name
					"key_name": schema.StringAttribute{Optional: true, Description: "GCP: resource name. Azure: Key Vault key name."},
					// Azure
					"tenant_id": schema.StringAttribute{Optional: true, Description: "Azure only."},
					"vault_uri": schema.StringAttribute{Optional: true, Description: "Azure only — Key Vault URI."},
					"client_id": schema.StringAttribute{Optional: true, Description: "Azure only — federated identity client ID."},
				},
			},
		},
	}
}

func (r *ExternalKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *ExternalKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExternalKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.CreateExternalKeyRequest{
		DisplayName:    plan.DisplayName.ValueString(),
		ProviderConfig: providerConfigFromModel(plan.ProviderConfig),
		Geo:            plan.Geo.ValueString(),
	}
	k, err := r.client.CreateExternalKey(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create external key", err.Error())
		return
	}
	externalKeyToModel(k, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ExternalKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ExternalKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := r.client.GetExternalKey(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read external key", err.Error())
		return
	}
	externalKeyToModel(k, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ExternalKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ExternalKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.UpdateExternalKeyRequest{}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		v := plan.DisplayName.ValueString()
		in.DisplayName = &v
	}
	if !plan.Geo.IsNull() && !plan.Geo.IsUnknown() {
		v := plan.Geo.ValueString()
		in.Geo = &v
	}
	if plan.ProviderConfig != nil {
		pc := providerConfigFromModel(plan.ProviderConfig)
		in.ProviderConfig = &pc
	}
	k, err := r.client.UpdateExternalKey(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update external key", err.Error())
		return
	}
	externalKeyToModel(k, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ExternalKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ExternalKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteExternalKey(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete external key", err.Error()+"\n\nNote: the API rejects deletion if any workspace still references this key. Remove `external_key_id` from referencing workspaces first.")
	}
}

func (r *ExternalKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Use types to silence unused-import lints when the resource file evolves.
var _ = types.StringNull
