package provider_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestPlanShape_UpdateInPlace(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("update_in_place_test", "Original Name"),
			},
			{
				Config: h.config("update_in_place_test", "Updated Name"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("supermetal_connector.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestPlanShape_UpdateInPlace_AddTable(t *testing.T) {
	h := newTestHarness(t)

	oneTable := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              orders = {}
            }
          }
        }`)

	twoTables := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              orders    = {}
              customers = {}
            }
          }
        }`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("add_table_test", "Add Table Test", oneTable),
			},
			{
				Config: h.config("add_table_test", "Add Table Test", twoTables),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("supermetal_connector.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestPlanShape_Replace(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("replace_test", "Replace Test"),
			},
			{
				Config: h.config("replace_test", "Replace Test", withDatabase("different_db")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("supermetal_connector.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}

func TestPlanShape_SensitiveMasking(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("sensitive_test", "Sensitive Test", withPassword("original-secret")),
			},
			{
				Config: h.config("sensitive_test", "Sensitive Test", withPassword("rotated-secret")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("supermetal_connector.test", plancheck.ResourceActionUpdate),
						plancheck.ExpectSensitiveValue("supermetal_connector.test",
							tfjsonpath.New("source").AtMapKey("postgres").AtMapKey("password")),
					},
				},
			},
		},
	})
}

func TestPlanShape_Composition(t *testing.T) {
	h := newTestHarness(t)

	config := h.config("composition_test", "Composition Test",
		withPasswordExpr("random_password.db.result"),
		withExtra(`resource "random_password" "db" {
  length  = 16
  special = false
}`))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {Source: "hashicorp/random", VersionConstraint: "~> 3.0"},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("supermetal_connector.test", "source.postgres.password"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestPlanShape_RefreshOnly(t *testing.T) {
	h := newTestHarness(t)

	config := h.config("refresh_only_test", "Refresh Only Test")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				RefreshState: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestPlanShape_ScaleSmoke(t *testing.T) {
	h := newTestHarness(t)

	var tables strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&tables, "              table_%03d = {}\n", i)
	}

	catalog := withCatalog(fmt.Sprintf(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
%s            }
          }
        }`, tables.String()))

	config := h.config("scale_smoke_test", "Scale Smoke Test", catalog)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
