package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider into the
// acceptance test framework. Acceptance tests only run when TF_ACC=1; the
// framework handles the env-var gate via resource.Test.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"anthropic": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck guards acceptance tests that hit the real Admin API.
// Called as PreCheck inside each acceptance test; the framework skips the
// step entirely when TF_ACC is unset, so this only fires for real runs.
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(envAdminAPIKey); v == "" {
		t.Fatalf("%s must be set for acceptance tests", envAdminAPIKey)
	}
}
