package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestValidateConfig_missingSource(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "test"
  name = "test"
  source = {}
  sink = {
    duckdb = {
      target_database = "main"
      connection = { quack = { url = "http://localhost:9494" } }
    }
  }
}`,
				ExpectError: regexp.MustCompile(`Missing source configuration`),
			},
		},
	})
}

func TestValidateConfig_missingSink(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "test"
  name = "test"
  source = {
    postgres = {
      host     = "localhost"
      port     = 5432
      database = "db"
      user     = "u"
      password = "p"
      ssl_mode = "Disable"
      replication_type = { snapshot = {} }
    }
  }
  sink = {}
}`,
				ExpectError: regexp.MustCompile(`Missing sink configuration`),
			},
		},
	})
}

func TestValidateConfig_snowflakeAuthBothSet(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "test"
  name = "test"
  source = {
    postgres = {
      host     = "localhost"
      port     = 5432
      database = "db"
      user     = "u"
      password = "p"
      ssl_mode = "Disable"
      replication_type = { snapshot = {} }
    }
  }
  sink = {
    snowflake = {
      account_identifier = "acct"
      user               = "u"
      warehouse          = "WH"
      target_database    = "DB"
      auth = {
        password = { password = "p" }
        key_pair = {
          private_key_pem = "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
        }
      }
    }
  }
}`,
				ExpectError: regexp.MustCompile(`Multiple Snowflake auth types specified`),
			},
		},
	})
}

func TestValidateConfig_snowflakeAuthNoneSet(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "test"
  name = "test"
  source = {
    postgres = {
      host     = "localhost"
      port     = 5432
      database = "db"
      user     = "u"
      password = "p"
      ssl_mode = "Disable"
      replication_type = { snapshot = {} }
    }
  }
  sink = {
    snowflake = {
      account_identifier = "acct"
      user               = "u"
      warehouse          = "WH"
      target_database    = "DB"
      auth = {}
    }
  }
}`,
				ExpectError: regexp.MustCompile(`Missing Snowflake auth configuration`),
			},
		},
	})
}

func TestValidateConfig_sshAuthBothSet(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "test"
  name = "test"
  source = {
    postgres = {
      host     = "localhost"
      port     = 5432
      database = "db"
      user     = "u"
      password = "p"
      ssl_mode = "Disable"
      replication_type = { snapshot = {} }
      tunnel = {
        ssh = {
          bastion_host = "bastion"
          bastion_port = 22
          user         = "tunnel"
          auth = {
            generated_key      = { private_key = "k1", public_key = "k2" }
            bring_your_own_key = { private_key = "key" }
          }
        }
      }
    }
  }
  sink = {
    duckdb = {
      target_database = "main"
      connection = { quack = { url = "http://localhost:9494" } }
    }
  }
}`,
				ExpectError: regexp.MustCompile(`Multiple SSH auth types specified`),
			},
		},
	})
}

func TestValidateConfig_sshAuthNoneSet(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "test"
  name = "test"
  source = {
    postgres = {
      host     = "localhost"
      port     = 5432
      database = "db"
      user     = "u"
      password = "p"
      ssl_mode = "Disable"
      replication_type = { snapshot = {} }
      tunnel = {
        ssh = {
          bastion_host = "bastion"
          bastion_port = 22
          user         = "tunnel"
          auth         = {}
        }
      }
    }
  }
  sink = {
    duckdb = {
      target_database = "main"
      connection = { quack = { url = "http://localhost:9494" } }
    }
  }
}`,
				ExpectError: regexp.MustCompile(`Missing SSH auth configuration`),
			},
		},
	})
}

func TestValidateConfig_connectorIDTooLong(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "supermetal" { endpoint = "http://localhost:3000" }
resource "supermetal_connector" "test" {
  id   = "this-id-is-way-too-long-for-the-thirty-character-limit"
  name = "test"
  source = {
    postgres = {
      host     = "localhost"
      port     = 5432
      database = "db"
      user     = "u"
      password = "p"
      ssl_mode = "Disable"
      replication_type = { snapshot = {} }
    }
  }
  sink = {
    duckdb = {
      target_database = "main"
      connection = { quack = { url = "http://localhost:9494" } }
    }
  }
}`,
				ExpectError: regexp.MustCompile(`string length must be at most 30`),
			},
		},
	})
}
