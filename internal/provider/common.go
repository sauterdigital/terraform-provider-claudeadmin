package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sauterdigital/terraform-provider-claude-admin/internal/anthropic"
)

// clientFromProviderData unwraps the provider's shared client. Resources and
// data sources receive ProviderData lazily, so a nil value is normal and not
// an error — the caller should just return and let configure run again later.
func clientFromProviderData(providerData any) (*anthropic.Client, diag.Diagnostics) {
	var diags diag.Diagnostics
	if providerData == nil {
		return nil, diags
	}
	c, ok := providerData.(*anthropic.Client)
	if !ok {
		diags.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *anthropic.Client, got: %T. Please open an issue.", providerData),
		)
		return nil, diags
	}
	return c, diags
}

func stringPointer(v string) *string { return &v }

func optionalStringValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func stringValueOrEmpty(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// nullableObject returns ObjectNull when v is nil; useful for nested API
// objects that the response may omit.
func nullableObject(attrs map[string]attr.Type, v map[string]attr.Value) (types.Object, diag.Diagnostics) {
	if v == nil {
		return types.ObjectNull(attrs), nil
	}
	return types.ObjectValue(attrs, v)
}
