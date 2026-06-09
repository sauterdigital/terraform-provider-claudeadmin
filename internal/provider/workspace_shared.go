package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-anthropic/internal/anthropic"
)

var dataResidencyAttrTypes = map[string]attr.Type{
	"workspace_geo":          types.StringType,
	"default_inference_geo":  types.StringType,
	"allowed_inference_geos": types.ListType{ElemType: types.StringType},
}

// dataResidencyObjectValue produces the types.Object that mirrors the API's
// data_residency block. We collapse the API's polymorphic
// allowed_inference_geos (either "unrestricted" or []string) onto a single
// list of strings, where ["unrestricted"] denotes unrestricted.
func dataResidencyObjectValue(ctx context.Context, dr *anthropic.DataResidency) (types.Object, diag.Diagnostics) {
	if dr == nil {
		return types.ObjectNull(dataResidencyAttrTypes), nil
	}
	geos, diags := types.ListValueFrom(ctx, types.StringType, dr.AllowedInferenceGeos.Values)
	if diags.HasError() {
		return types.ObjectNull(dataResidencyAttrTypes), diags
	}
	return types.ObjectValue(dataResidencyAttrTypes, map[string]attr.Value{
		"workspace_geo":          types.StringValue(dr.WorkspaceGeo),
		"default_inference_geo":  types.StringValue(dr.DefaultInferenceGeo),
		"allowed_inference_geos": geos,
	})
}

func tagsMapValue(ctx context.Context, tags map[string]string) (types.Map, diag.Diagnostics) {
	if tags == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, tags)
}

func tagsFromTerraform(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]string{}
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// dataResidencyFromTerraform converts the Terraform data_residency object back
// to the API shape, collapsing the list-of-strings convention back onto the
// API's polymorphic allowed_inference_geos (string OR array). Returns nil if
// the object is null/unknown so we omit the field from API requests entirely.
func dataResidencyFromTerraform(ctx context.Context, obj types.Object) (*anthropic.DataResidency, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	attrs := obj.Attributes()
	dr := &anthropic.DataResidency{}
	if v, ok := attrs["workspace_geo"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		dr.WorkspaceGeo = v.ValueString()
	}
	if v, ok := attrs["default_inference_geo"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		dr.DefaultInferenceGeo = v.ValueString()
	}
	if v, ok := attrs["allowed_inference_geos"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
		var geos []string
		d := v.ElementsAs(ctx, &geos, false)
		diags.Append(d...)
		dr.AllowedInferenceGeos.Values = geos
	}
	return dr, diags
}
