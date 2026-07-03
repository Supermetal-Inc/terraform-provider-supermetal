package provider_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccConnectorResource_basic(t *testing.T) {
	h := newTestHarness(t)

	config := h.config("test-pg-duckdb", "Test Connector")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "id", "test-pg-duckdb"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "name", "Test Connector"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "disabled", "false"),
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

func TestAccConnectorResource_credentialRotation(t *testing.T) {
	h := newTestHarness(t)

	configOriginal := h.config("test-cred-rotation", "Credential Rotation Test",
		withPassword("original-password"))
	configRotated := h.config("test-cred-rotation", "Credential Rotation Test",
		withPassword("new-rotated-password"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configOriginal,
			},
			{
				Config: configRotated,
			},
			{
				Config: configRotated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccConnectorResource_externalDrift(t *testing.T) {
	h := newTestHarness(t)

	config := h.config("test-drift", "Drift Test")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				PreConfig: func() {
					t.Log("PreConfig: modifying connector via API...")
					modifiedConnector := fmt.Sprintf(`{
  "id": "test-drift",
  "name": "Drift Test - MODIFIED EXTERNALLY",
  "source": {
    "postgres": {
      "connection": {
        "host": "%s",
        "port": %d,
        "database": "testdb",
        "user": "testuser",
        "password": "testpass",
        "ssl_mode": "Disable"
      },
      "replication_type": {
        "snapshot": {}
      }
    }
  },
  "sink": {
    "duckdb": {
      "target_database": "main",
      "connection": {
        "quack": {
          "url": "http://localhost:9494",
          "ssl": false
        }
      }
    }
  }
}`, h.postgresHost, h.postgresPort)

					req, err := http.NewRequest("POST", h.agentEndpoint+"/api/v1/connectors/test-drift", strings.NewReader(modifiedConnector))
					if err != nil {
						t.Fatalf("Failed to build request: %v", err)
					}
					req.Header.Set("Content-Type", "application/json")
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						t.Fatalf("Failed to modify connector externally: %v", err)
					}
					_ = resp.Body.Close()
					if resp.StatusCode != 200 && resp.StatusCode != 201 {
						t.Fatalf("Failed to modify connector externally: status %d", resp.StatusCode)
					}
					t.Logf("PreConfig: connector modified successfully, status %d", resp.StatusCode)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
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

func TestAccConnectorResource_import(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("test-import", "Import Test"),
			},
			{
				ResourceName:      "supermetal_connector.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"source.postgres.password",
					"sink.duckdb.connection.quack.token",
				},
			},
		},
	})
}

func TestAccConnectorResource_tableSelection(t *testing.T) {
	h := newTestHarness(t)

	catalogOneTable := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            action = "Include"
            tables = {
              users = { action = "Include" }
            }
          }
        }`)

	catalogTwoTables := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            action = "Include"
            tables = {
              users  = { action = "Include" }
              orders = { action = "Include" }
            }
          }
        }`)

	catalogOrdersOnly := withCatalog(`        name           = "testdb"
        default_action = "Exclude"
        schemas = {
          public = {
            action = "Include"
            tables = {
              orders = { action = "Include" }
            }
          }
        }`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("test-table-selection", "Table Selection Test", catalogOneTable),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.name", "testdb"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.default_action", "Exclude"),
				),
			},
			{
				Config: h.config("test-table-selection", "Table Selection Test", catalogOneTable),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: h.config("test-table-selection", "Table Selection Test", catalogTwoTables),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.schemas.public.tables.%", "2"),
				),
			},
			{
				Config: h.config("test-table-selection", "Table Selection Test", catalogTwoTables),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: h.config("test-table-selection", "Table Selection Test", catalogOrdersOnly),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.schemas.public.tables.%", "1"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.schemas.public.tables.orders.action", "Include"),
				),
			},
			{
				Config: h.config("test-table-selection", "Table Selection Test", catalogOrdersOnly),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccConnectorResource_validation(t *testing.T) {
	h := newTestHarness(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      h.config("test-validation", "Validation Test", withValidation()),
				ExpectError: regexp.MustCompile(`Sink validation failed`),
			},
			{
				Config: h.config("test-validation-disabled", "Validation Disabled Test",
					withValidation(), withDisabled()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "disabled", "true"),
				),
			},
		},
	})
}

func TestAccConnectorResource_catalogOverlayMode(t *testing.T) {
	h := newTestHarness(t)

	config := h.config("test-catalog-overlay", "Catalog Overlay Test", withCatalog(`        name           = "testdb"
        default_action = "Include"
        schemas = {
          public = {
            action = "Exclude"
            tables = {}
          }
        }`))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.default_action", "Include"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.schemas.public.action", "Exclude"),
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

func TestAccConnectorResource_catalogDefaultsOmitted(t *testing.T) {
	h := newTestHarness(t)

	config := h.config("test-catalog-defaults", "Catalog Defaults Test", withCatalog(`        name = "testdb"
        schemas = {
          public = {
            tables = {
              orders = {}
            }
          }
        }`))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.default_action", "Exclude"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.schemas.public.action", "Include"),
					resource.TestCheckResourceAttr("supermetal_connector.test", "source.postgres.catalog.schemas.public.tables.orders.action", "Include"),
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

func TestAccConnectorResource_catalogMutationDrift(t *testing.T) {
	h := newTestHarness(t)

	catalogEmpty := withCatalog(`        name = "testdb"
        schemas = {
          public = {
            tables = {}
          }
        }`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: h.config("test-catalog-drift", "Catalog Drift Test Round 1", catalogEmpty),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "name", "Catalog Drift Test Round 1"),
				),
			},
			{
				Config: h.config("test-catalog-drift", "Catalog Drift Test Round 2", catalogEmpty),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "name", "Catalog Drift Test Round 2"),
				),
			},
			{
				Config: h.config("test-catalog-drift", "Catalog Drift Test Round 3", catalogEmpty),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("supermetal_connector.test", "name", "Catalog Drift Test Round 3"),
				),
			},
			{
				Config: h.config("test-catalog-drift", "Catalog Drift Test Round 3", catalogEmpty),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
