package accesscontextmanager_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
	"github.com/hashicorp/terraform-provider-google/google/envvar"
	"github.com/hashicorp/terraform-provider-google/google/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"
)

// Since each test here is acting on the same organization and only one AccessPolicy
// can exist, they need to be run serially. See AccessPolicy for the test runner.

func testAccAccessContextManagerServicePerimeterDryRunEgressPolicy_basicTest(t *testing.T) {
	org := envvar.GetTestOrgFromEnv(t)
	//projects := acctest.BootstrapServicePerimeterProjects(t, 1)
	policyTitle := acctest.RandString(t, 10)
	perimeterTitle := "perimeter"
	projectNumber := envvar.GetTestProjectNumberFromEnv()

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testAccAccessContextManagerServicePerimeterDryRunEgressPolicy_basic(org, policyTitle, perimeterTitle, projectNumber),
			},
			{
				Config: testAccAccessContextManagerServicePerimeterDryRunEgressPolicy_destroy(org, policyTitle, perimeterTitle),
				Check:  testAccCheckAccessContextManagerServicePerimeterDryRunEgressPolicyDestroyProducer(t),
			},
		},
	})
}

func testAccCheckAccessContextManagerServicePerimeterDryRunEgressPolicyDestroyProducer(t *testing.T) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "google_access_context_manager_service_perimeter_dry_run_egress_policy" {
				continue
			}

			config := acctest.GoogleProviderConfig(t)

			url, err := tpgresource.ReplaceVarsForTest(config, rs, "{{AccessContextManagerBasePath}}{{perimeter}}")
			if err != nil {
				return err
			}

			res, err := transport_tpg.SendRequest(transport_tpg.SendRequestOptions{
				Config:    config,
				Method:    "GET",
				RawURL:    url,
				UserAgent: config.UserAgent,
			})
			if err != nil {
				return err
			}

			v, ok := res["spec"]
			if !ok || v == nil {
				return nil
			}

			res = v.(map[string]interface{})
			v, ok = res["egress_policies"]
			if !ok || v == nil {
				return nil
			}

			resources := v.([]interface{})
			if len(resources) == 0 {
				return nil
			}

			return fmt.Errorf("expected 0 resources in perimeter, found %d: %v", len(resources), resources)
		}

		return nil
	}
}

func testAccAccessContextManagerServicePerimeterDryRunEgressPolicy_basic(org, policyTitle, perimeterTitleName, projectNumber string) string {
	return fmt.Sprintf(`
%s

resource "google_access_context_manager_access_level" "test-access" {
  parent      = "accessPolicies/${google_access_context_manager_access_policy.test-access.name}"
  name        = "accessPolicies/${google_access_context_manager_access_policy.test-access.name}/accessLevels/level"
  title       = "level"
  description = "hello"
  basic {
    combining_function = "AND"
    conditions {
      ip_subnetworks = ["192.0.4.0/24"]
    }
  }
}

resource "google_access_context_manager_service_perimeter_dry_run_egress_policy" "test-access1" {
  perimeter = google_access_context_manager_service_perimeter.test-access.name
	title = "egress policy title 1"
	egress_from {
		identity_type = "ANY_USER_ACCOUNT"
	}
	egress_to {
	  operations {
		service_name = "storage.googleapis.com"
		method_selectors {
		  method = "*"
	    }
	  }
	}
}

resource "google_access_context_manager_service_perimeter_dry_run_egress_policy" "test-access2" {
	perimeter = google_access_context_manager_service_perimeter.test-access.name
	title = "egress policy title 2"
	egress_from {
		identity_type = "ANY_USER_ACCOUNT"
		sources {
			access_level = google_access_context_manager_access_level.test-access.name
		}
		source_restriction = "SOURCE_RESTRICTION_ENABLED"
	}
	egress_to {
		resources = ["*"]
		roles = ["roles/bigquery.admin"]
	}
  	depends_on = [google_access_context_manager_service_perimeter_dry_run_egress_policy.test-access1]
}

resource "google_access_context_manager_service_perimeter_dry_run_egress_policy" "test-access3" {
  perimeter = google_access_context_manager_service_perimeter.test-access.name
	egress_from {
		sources {
			resource = "projects/%s"
		}
		source_restriction = "SOURCE_RESTRICTION_ENABLED"
	}
}

`, testAccAccessContextManagerServicePerimeterDryRunEgressPolicy_destroy(org, policyTitle, perimeterTitleName), projectNumber)
}

func testAccAccessContextManagerServicePerimeterDryRunEgressPolicy_destroy(org, policyTitle, perimeterTitleName string) string {
	return fmt.Sprintf(`
resource "google_access_context_manager_access_policy" "test-access" {
  parent = "organizations/%s"
  title  = "%s"
}

resource "google_access_context_manager_service_perimeter" "test-access" {
  parent         = "accessPolicies/${google_access_context_manager_access_policy.test-access.name}"
  name           = "accessPolicies/${google_access_context_manager_access_policy.test-access.name}/servicePerimeters/%s"
  title          = "%s"
  spec {
    restricted_services = ["storage.googleapis.com"]
  }
  use_explicit_dry_run_spec = true

  lifecycle {
  	ignore_changes = [spec[0].egress_policies]
  }
}
`, org, policyTitle, perimeterTitleName, perimeterTitleName)
}
