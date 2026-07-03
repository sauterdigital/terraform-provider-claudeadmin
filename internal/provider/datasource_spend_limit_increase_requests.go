package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

// ---------- Single (by id) ----------

var (
	_ datasource.DataSource              = &SpendLimitIncreaseRequestDataSource{}
	_ datasource.DataSourceWithConfigure = &SpendLimitIncreaseRequestDataSource{}
)

func NewSpendLimitIncreaseRequestDataSource() datasource.DataSource {
	return &SpendLimitIncreaseRequestDataSource{}
}

type SpendLimitIncreaseRequestDataSource struct{ client *anthropic.Client }

type IncreaseRequestModel struct {
	ID          types.String `tfsdk:"id"`
	Status      types.String `tfsdk:"status"`
	ActorUserID types.String `tfsdk:"actor_user_id"`
	ActorEmail  types.String `tfsdk:"actor_email"`
	ActorName   types.String `tfsdk:"actor_name"`
	Amount      types.String `tfsdk:"amount"`
	Currency    types.String `tfsdk:"currency"`
	Period      types.String `tfsdk:"period"`
	CreatedAt   types.String `tfsdk:"created_at"`
	ResolvedAt  types.String `tfsdk:"resolved_at"`
	ResolvedBy  types.String `tfsdk:"resolved_by"`
	Description types.String `tfsdk:"description"`
}

func (d *SpendLimitIncreaseRequestDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spend_limit_increase_request"
}

func (d *SpendLimitIncreaseRequestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single spend limit increase request by ID.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Required: true},
			"status":        schema.StringAttribute{Computed: true},
			"actor_user_id": schema.StringAttribute{Computed: true},
			"actor_email":   schema.StringAttribute{Computed: true},
			"actor_name":    schema.StringAttribute{Computed: true},
			"amount":        schema.StringAttribute{Computed: true},
			"currency":      schema.StringAttribute{Computed: true},
			"period":        schema.StringAttribute{Computed: true},
			"created_at":    schema.StringAttribute{Computed: true},
			"resolved_at":   schema.StringAttribute{Computed: true},
			"resolved_by":   schema.StringAttribute{Computed: true, Description: "User email or scoped API key ID that resolved the request."},
			"description":   schema.StringAttribute{Computed: true},
		},
	}
}

func (d *SpendLimitIncreaseRequestDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *SpendLimitIncreaseRequestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IncreaseRequestModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r, err := d.client.GetSpendLimitIncreaseRequest(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read increase request", err.Error())
		return
	}
	increaseRequestToModel(r, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// ---------- List ----------

var (
	_ datasource.DataSource              = &SpendLimitIncreaseRequestsDataSource{}
	_ datasource.DataSourceWithConfigure = &SpendLimitIncreaseRequestsDataSource{}
)

func NewSpendLimitIncreaseRequestsDataSource() datasource.DataSource {
	return &SpendLimitIncreaseRequestsDataSource{}
}

type SpendLimitIncreaseRequestsDataSource struct{ client *anthropic.Client }

type IncreaseRequestsListModel struct {
	ActorIDs []types.String         `tfsdk:"actor_ids"`
	Status   []types.String         `tfsdk:"status"`
	Requests []IncreaseRequestModel `tfsdk:"requests"`
}

func (d *SpendLimitIncreaseRequestsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spend_limit_increase_requests"
}

func (d *SpendLimitIncreaseRequestsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists spend limit increase requests, most recent first.",
		Attributes: map[string]schema.Attribute{
			"actor_ids": schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Filter by requester user IDs."},
			"status": schema.ListAttribute{
				Optional: true, ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("pending", "approved", "denied")),
				},
			},
			"requests": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"status":        schema.StringAttribute{Computed: true},
						"actor_user_id": schema.StringAttribute{Computed: true},
						"actor_email":   schema.StringAttribute{Computed: true},
						"actor_name":    schema.StringAttribute{Computed: true},
						"amount":        schema.StringAttribute{Computed: true},
						"currency":      schema.StringAttribute{Computed: true},
						"period":        schema.StringAttribute{Computed: true},
						"created_at":    schema.StringAttribute{Computed: true},
						"resolved_at":   schema.StringAttribute{Computed: true},
						"resolved_by":   schema.StringAttribute{Computed: true},
						"description":   schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *SpendLimitIncreaseRequestsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.client = c
}

func (d *SpendLimitIncreaseRequestsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IncreaseRequestsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.client.ListSpendLimitIncreaseRequests(ctx, anthropic.ListIncreaseRequestsParams{
		ActorIDs: stringSliceFromTF(data.ActorIDs),
		Status:   stringSliceFromTF(data.Status),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list increase requests", err.Error())
		return
	}
	out := make([]IncreaseRequestModel, 0, len(list))
	for i := range list {
		var m IncreaseRequestModel
		increaseRequestToModel(&list[i], &m)
		out = append(out, m)
	}
	data.Requests = out
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func increaseRequestToModel(r *anthropic.SpendLimitIncreaseRequest, m *IncreaseRequestModel) {
	m.ID = types.StringValue(r.ID)
	m.Status = types.StringValue(r.Status)
	m.ActorUserID = stringValueOrEmpty(r.Actor.UserID)
	m.ActorEmail = stringValueOrEmpty(r.Actor.EmailAddress)
	m.ActorName = stringValueOrEmpty(r.Actor.Name)
	m.Amount = stringValueOrEmpty(r.Amount)
	m.Currency = stringValueOrEmpty(r.Currency)
	m.Period = types.StringValue(r.Period)
	m.CreatedAt = types.StringValue(r.CreatedAt)
	m.ResolvedAt = stringValueOrEmpty(r.ResolvedAt)
	resolvedBy := ""
	if r.ResolvedBy != nil {
		if r.ResolvedBy.EmailAddress != "" {
			resolvedBy = r.ResolvedBy.EmailAddress
		} else if r.ResolvedBy.ScopedAPIKeyID != "" {
			resolvedBy = r.ResolvedBy.ScopedAPIKeyID
		}
	}
	m.ResolvedBy = stringValueOrEmpty(resolvedBy)
	m.Description = stringValueOrEmpty(r.Description)
}
