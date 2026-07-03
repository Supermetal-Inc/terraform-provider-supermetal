# Deploy a connector without starting it.
# Useful for pre-staging config that will be enabled later.

resource "supermetal_connector" "staged" {
  id       = "staged-connector"
  name     = "Staged pipeline"
  disabled = true

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Disable"

      replication_type = { snapshot = {} }
    }
  }

  sink = {
    duckdb = {
      target_database = "warehouse"
      connection      = { quack = { url = var.duckdb_url } }
    }
  }
}

# terraform apply                       creates the connector but does not start it
# set disabled = false, terraform apply  enables the connector
