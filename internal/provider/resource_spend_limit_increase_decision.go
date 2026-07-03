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
	_ resource.Resource                = &SpendLimitIncreaseDecisionResource{}
	_ resource.ResourceWithConfigure   = &SpendLimitIncreaseDecisionResource{}
	_ resource.ResourceWithImportState = &SpendLimitIncreaseDecisionResource{}
)

func NewSpendLimitIncreaseDecisionResource() resource.Resource {
	return &SpendLimitIncreaseDecisionResource{}
}

type SpendLimitIncreaseDecisionResource struct{ client *anthropic.Client }

type IncreaseDecisionModel struct {
	ID                   types.String `tfsdk:"id"`
	RequestID            types.String `tfsdk:"request_id"`
	Decision             types.String `tfsdk:"decision"`
	Amount               types.String `tfsdk:"amount"`
	Period               types.String `tfsdk:"period"`
	SuppressNotification types.Bool   `tfsdk:"suppress_notification"`
	Status               types.String `tfsdk:"status"`
	ResolvedAt           types.String `tfsdk:"resolved_at"`
}

func (r *SpendLimitIncreaseDecisionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spend_limit_increase_decision"
}

func (r *SpendLimitIncreaseDecisionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Records a decision (approve / deny) on a spend limit increase request. Decisions are one-way and immutable — any change to `decision`, `amount`, or `period` forces replacement (which in practice will fail because the request is already resolved). Use `terraform destroy` to remove the decision from state without affecting the underlying request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Same as `request_id`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"request_id": schema.StringAttribute{
				Description:   "Increase request ID to act on.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"decision": schema.StringAttribute{
				Description:   "Either `approve` or `deny`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("approve", "deny"),
				},
			},
			"amount": schema.StringAttribute{
				Description:   "Required when decision is `approve`. Non-negative integer decimal string in minor currency units.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"period": schema.StringAttribute{
				Description:   "Period for the approved limit. Defaults to the period the request was made on.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("monthly", "daily", "weekly"),
				},
			},
			"suppress_notification": schema.BoolAttribute{
				Description:   "If true, the requester is NOT emailed.",
				Optional:      true,
				PlanModifiers: []planmodifier.Bool{},
			},
			"status":      schema.StringAttribute{Computed: true},
			"resolved_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *SpendLimitIncreaseDecisionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.client = c
}

func (r *SpendLimitIncreaseDecisionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IncreaseDecisionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result *anthropic.SpendLimitIncreaseRequest
	var err error
	switch plan.Decision.ValueString() {
	case "approve":
		if plan.Amount.IsNull() || plan.Amount.ValueString() == "" {
			resp.Diagnostics.AddError("Missing amount", "`amount` is required when decision = approve.")
			return
		}
		result, err = r.client.ApproveSpendLimitIncreaseRequest(ctx, plan.RequestID.ValueString(), anthropic.ApproveIncreaseRequest{
			Amount:               plan.Amount.ValueString(),
			Period:               plan.Period.ValueString(),
			SuppressNotification: plan.SuppressNotification.ValueBool(),
		})
	case "deny":
		result, err = r.client.DenySpendLimitIncreaseRequest(ctx, plan.RequestID.ValueString(), anthropic.DenyIncreaseRequest{
			SuppressNotification: plan.SuppressNotification.ValueBool(),
		})
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to record decision", err.Error())
		return
	}
	plan.ID = types.StringValue(result.ID)
	plan.Status = types.StringValue(result.Status)
	plan.ResolvedAt = stringValueOrEmpty(result.ResolvedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *SpendLimitIncreaseDecisionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IncreaseDecisionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.GetSpendLimitIncreaseRequest(ctx, state.RequestID.ValueString())
	if err != nil {
		if anthropic.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read increase request", err.Error())
		return
	}
	state.Status = types.StringValue(result.Status)
	state.ResolvedAt = stringValueOrEmpty(result.ResolvedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *SpendLimitIncreaseDecisionResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All mutable attributes are RequireReplace; Update is unreachable.
}

func (r *SpendLimitIncreaseDecisionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Decisions are not reversible at the API. Removing from state is a no-op.
}

func (r *SpendLimitIncreaseDecisionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("request_id"), req, resp)
}
