# Supermetal Terraform Provider

Terraform provider for managing [Supermetal](https://supermetal.io) CDC connectors.

## Documentation

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/supermetal-inc/supermetal/latest/docs).

## Quick start

```hcl
terraform {
  required_providers {
    supermetal = {
      source  = "supermetal-inc/supermetal"
      version = "~> 0.1"
    }
  }
}

provider "supermetal" {
  endpoint = "https://supermetal.internal:3000"
}

resource "supermetal_connector" "orders" {
  id   = "orders-to-warehouse"
  name = "Orders pipeline"

  source = {
    postgres = {
      host     = "postgres.internal"
      port     = 5432
      database = "production"
      user     = "replicator"
      password = var.pg_password
      ssl_mode = "Require"

      replication_type = {
        logical_replication = {
          publication_name = "supermetal_orders"
        }
      }
    }
  }

  sink = {
    snowflake = {
      account_identifier = "xy12345.us-east-1"
      user               = "SUPERMETAL"
      warehouse          = "COMPUTE_WH"
      target_database    = "ANALYTICS"

      auth = {
        key_pair = {
          private_key_pem = var.snowflake_private_key
        }
      }
    }
  }
}
```

## Code generation

Most of this provider is generated from the Supermetal agent's OpenAPI spec by an internal generator. Regeneration is vendor-side. Generated files carry `DO NOT EDIT` headers stamped with the exact generator commit and spec version they were built from.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- A running [Supermetal agent](https://docs.supermetal.io/docs/main/concepts/deployment)

## Development

```bash
make build      # compile the provider
make test       # unit tests
make testacc    # acceptance tests (requires SUPERMETAL_AGENT_BINARY)
make verify     # full verification suite
```

## License

[Apache License 2.0](LICENSE)
