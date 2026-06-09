package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccWorkspace_basic exercises the workspace resource's full lifecycle
// against the live Admin API: create with a name, update the name, then
// import to verify ImportState round-trips state cleanly.
//
// Requires TF_ACC=1 and ANTHROPIC_ADMIN_API_KEY. Creates and archives one
// real workspace — runs cost money and mutate org state.
func TestAccWorkspace_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	updated := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anthropic_workspace.test", "name", name),
					resource.TestCheckResourceAttrSet("anthropic_workspace.test", "id"),
					resource.TestCheckResourceAttrSet("anthropic_workspace.test", "created_at"),
					resource.TestCheckResourceAttrSet("anthropic_workspace.test", "display_color"),
				),
			},
			{
				Config: testAccWorkspaceConfig(updated),
				Check:  resource.TestCheckResourceAttr("anthropic_workspace.test", "name", updated),
			},
			{
				ResourceName:      "anthropic_workspace.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccWorkspace_withTags verifies the tags map round-trips through the
// API. Tag updates exercise the same Update path and prove the omitempty
// behavior in CreateWorkspaceRequest doesn't drop user-supplied tags.
func TestAccWorkspace_withTags(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-tags")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceConfigWithTags(name, map[string]string{"env": "test", "team": "platform"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anthropic_workspace.tagged", "tags.env", "test"),
					resource.TestCheckResourceAttr("anthropic_workspace.tagged", "tags.team", "platform"),
				),
			},
			{
				Config: testAccWorkspaceConfigWithTags(name, map[string]string{"env": "staging"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anthropic_workspace.tagged", "tags.env", "staging"),
					resource.TestCheckNoResourceAttr("anthropic_workspace.tagged", "tags.team"),
				),
			},
		},
	})
}

func testAccWorkspaceConfig(name string) string {
	return fmt.Sprintf(`
resource "anthropic_workspace" "test" {
  name = %q
}
`, name)
}

func testAccWorkspaceConfigWithTags(name string, tags map[string]string) string {
	pairs := ""
	for k, v := range tags {
		pairs += fmt.Sprintf("    %s = %q\n", k, v)
	}
	return fmt.Sprintf(`
resource "anthropic_workspace" "tagged" {
  name = %q
  tags = {
%s  }
}
`, name, pairs)
}
