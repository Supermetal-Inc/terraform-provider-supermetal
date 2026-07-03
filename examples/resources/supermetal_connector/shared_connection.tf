# Two connectors from the same database with different table selections
#
# Define shared connection fields once in locals. Each connector
# adds its own publication and catalog.

locals {
  pg = {
    host     = var.pg_host
    port     = 5432
    database = var.pg_database
    user     = var.pg_user
    password = var.pg_password
    ssl_mode = "Require"
  }
}

# Orders tables
resource "supermetal_connector" "orders" {
  id   = "orders-pipeline"
  name = "Orders"

  source = {
    postgres = {
      host     = local.pg.host
      port     = local.pg.port
      database = local.pg.database
      user     = local.pg.user
      password = local.pg.password
      ssl_mode = local.pg.ssl_mode

      replication_type = {
        logical_replication = { publication_name = "sm_orders" }
      }

      catalog = {
        name           = local.pg.database
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              orders      = {}
              order_items = {}
            }
          }
        }
      }
    }
  }

  sink = {
    duckdb = {
      target_database = "warehouse"
      connection      = { quack = { url = var.duckdb_url } }
    }
  }
}

# Customer tables (same source, different publication and catalog)
resource "supermetal_connector" "customers" {
  id   = "customers-pipeline"
  name = "Customers"

  source = {
    postgres = {
      host     = local.pg.host
      port     = local.pg.port
      database = local.pg.database
      user     = local.pg.user
      password = local.pg.password
      ssl_mode = local.pg.ssl_mode

      replication_type = {
        logical_replication = { publication_name = "sm_customers" }
      }

      catalog = {
        name           = local.pg.database
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              customers = {}
              addresses = {}
            }
          }
        }
      }
    }
  }

  sink = {
    duckdb = {
      target_database = "warehouse"
      connection      = { quack = { url = var.duckdb_url } }
    }
  }
}
