package provider_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestExamples(t *testing.T) {
	examples, err := filepath.Glob("../../examples/resources/supermetal_connector/*.tf")
	if err != nil {
		t.Fatal(err)
	}
	if len(examples) == 0 {
		t.Fatal("no example files found")
	}

	for _, path := range examples {
		name := strings.TrimSuffix(filepath.Base(path), ".tf")
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			cleaned := stripVariables(string(content))

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:             exampleVars + cleaned,
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				}},
			})
		})
	}
}

func stripVariables(content string) string {
	re := regexp.MustCompile(`(?s)variable\s+"[^"]+"\s*\{[^{}]*\}\s*`)
	return re.ReplaceAllString(content, "")
}

const exampleVars = `
provider "supermetal" {
  endpoint = "http://localhost:3000"
}

variable "pg_host" { default = "localhost" }
variable "pg_database" { default = "testdb" }
variable "pg_user" { default = "testuser" }
variable "pg_password" { default = "testpass" }
variable "pg_ssl_root_cert" { default = "-----BEGIN CERTIFICATE-----\nMIIB=\n-----END CERTIFICATE-----" }
variable "duckdb_url" { default = "http://localhost:9494" }
variable "snowflake_account" { default = "xy12345.us-east-1" }
variable "snowflake_user" { default = "testuser" }
variable "snowflake_password" { default = "testpass" }
variable "snowflake_private_key" { default = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBg=\n-----END PRIVATE KEY-----" }
variable "snowflake_key_password" { default = "keypass" }
variable "ssh_private_key" { default = "-----BEGIN OPENSSH PRIVATE KEY-----\nb3Blbg==\n-----END OPENSSH PRIVATE KEY-----" }
`
