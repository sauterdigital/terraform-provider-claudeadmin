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
	_ resource.Resource                = &SpendLimitResource{}
	_ resource.ResourceWithImportState = &SpendLimitResource{}
	_ resource.ResourceWithConfigure   = &SpendLimitResource{}
)

func NewSpendLimitResource() resource.Resource { return &SpendLimitResource{} }

type SpendLimitResource struct{ client *anthropic.Client }

type SpendLimitResourceModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.String `tfsdk:"user_id"`
	Amount    types.String `tfsdk:"amount"`
	Period    types.String `tfsdk:"period"`
	Currency  types.String `tfsdk:"currency"`
	ScopeType types.String `tfsdk:"scope_type"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *SpendLimitResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spend_limit"
}

func (r *SpendLimitResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PER-USER spend limit override. The Admin API only accepts `scope.type=user` for writes — seat-tier, rbac_group, organization_service, and organization-level limits are configured in claude.ai and surface here read-only via `anthropic_effective_spend_limits`. Amount is a decimal string in minor units (e.g. \"12345\" = $123.45 USD).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Description:   "User ID this limit applies to. Changing forces replacement (the upsert key is (scope, period)).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"amount": schema.StringAttribute{
				Description: "Non-negative integer decimal string in minor currency units (cents for USD).",
				Required:    true,
			},
			"period": schema.StringAttribute{
				Description:   "Billing period for this limit. Changing forces replacement.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.OneOf("monthly", "daily", "weekly"),
				},
			},
			"currency": schema.StringAttribute{
				Description: "Currency for the amount. Typically `USD`.",
				Computed:    true,
			},
			"scope_type": schema.StringAttribute{
				Description: "Always `user` for limits managed by this resource.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *SpendLimitResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *SpendLimitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SpendLimitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := anthropic.SetSpendLimitRequest{
		Amount: plan.Amount.ValueString(),
		Scope:  anthropic.SpendLimitScope{Type: "user", UserID: plan.UserID.ValueString()},
		Period: plan.Period.ValueString(),
	}
	sl, err := r.client.SetSpendLimit(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to set spend limit", err.Error())
		return
	}
	spendLimitToModel(sl, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *SpendLimitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SpendLimitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sl, err := r.client.GetSpendLimit(ctx, state.ID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read spend limit", err.Error())
		return
	}
	spendLimitToModel(sl, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *SpendLimitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SpendLimitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// API is upsert by (scope, period); re-Set updates in place.
	in := anthropic.SetSpendLimitRequest{
		Amount: plan.Amount.ValueString(),
		Scope:  anthropic.SpendLimitScope{Type: "user", UserID: plan.UserID.ValueString()},
		Period: plan.Period.ValueString(),
	}
	sl, err := r.client.SetSpendLimit(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update spend limit", err.Error())
		return
	}
	spendLimitToModel(sl, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *SpendLimitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SpendLimitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSpendLimit(ctx, state.ID.ValueString()); err != nil && !anthropic.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete spend limit", err.Error())
	}
}

func (r *SpendLimitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func spendLimitToModel(sl *anthropic.SpendLimit, m *SpendLimitResourceModel) {
	m.ID = types.StringValue(sl.ID)
	m.Amount = types.StringValue(sl.Amount)
	m.Period = types.StringValue(sl.Period)
	m.Currency = types.StringValue(sl.Currency)
	m.ScopeType = types.StringValue(sl.Scope.Type)
	m.UserID = types.StringValue(sl.Scope.UserID)
	m.CreatedAt = types.StringValue(sl.CreatedAt)
	m.UpdatedAt = types.StringValue(sl.UpdatedAt)
}
