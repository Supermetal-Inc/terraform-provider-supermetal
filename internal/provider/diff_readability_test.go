package provider_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

type logPlanCheck struct {
	t *testing.T
}

func (l logPlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		l.t.Logf("\n=== PLAN DIFF for %s ===", rc.Address)
		l.t.Logf("Action: %v", rc.Change.Actions)

		if rc.Change.Before != nil {
			before, _ := json.MarshalIndent(rc.Change.Before, "", "  ")
			l.t.Logf("Before:\n%s", string(before))
		}
		if rc.Change.After != nil {
			after, _ := json.MarshalIndent(rc.Change.After, "", "  ")
			l.t.Logf("After:\n%s", string(after))
		}
	}
}

func TestDiffReadability_CatalogTableAddition(t *testing.T) {
	h := newTestHarness(t)

	threeTableCatalog := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              users    = {}
              orders   = {}
              products = {}
            }
          }
        }`)

	fiveTableCatalog := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              users     = {}
              orders    = {}
              products  = {}
              inventory = {}
              shipments = {}
            }
          }
        }`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("diff-readability-test", "Diff Readability Test",
					withResourceName("diff_test"), threeTableCatalog),
			},
			{
				Config: h.config("diff-readability-test", "Diff Readability Test",
					withResourceName("diff_test"), fiveTableCatalog),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						logPlanCheck{t: t},
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("supermetal_connector.diff_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestDiffReadability_CatalogTableInsertMiddle(t *testing.T) {
	h := newTestHarness(t)

	threeTableCatalog := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              users    = {}
              orders   = {}
              products = {}
            }
          }
        }`)

	fourTableCatalog := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              users     = {}
              customers = {}
              orders    = {}
              products  = {}
            }
          }
        }`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("insert-middle-test", "Insert Middle Test",
					withResourceName("insert_test"), threeTableCatalog),
			},
			{
				Config: h.config("insert-middle-test", "Insert Middle Test",
					withResourceName("insert_test"), fourTableCatalog),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						logPlanCheck{t: t},
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("supermetal_connector.insert_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestDiffReadability_NameChange(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("name-change-test", "Original Name",
					withResourceName("name_test")),
			},
			{
				Config: h.config("name-change-test", "Updated Pipeline Name",
					withResourceName("name_test")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						logPlanCheck{t: t},
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("supermetal_connector.name_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestDiffReadability_PasswordRotation(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("password-rotation-test", "Password Test",
					withResourceName("password_test"), withPassword("original-password-123")),
			},
			{
				Config: h.config("password-rotation-test", "Password Test",
					withResourceName("password_test"), withPassword("rotated-password-456")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						logPlanCheck{t: t},
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("supermetal_connector.password_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}
