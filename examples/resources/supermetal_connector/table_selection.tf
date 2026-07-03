# Table selection patterns
#
# The catalog block controls which tables are replicated.
# default_action sets the baseline, then schemas and tables override.

# Pattern 1. Replicate everything except specific tables
resource "supermetal_connector" "include_by_default" {
  id   = "include-by-default"
  name = "Include by default"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Disable"

      replication_type = { snapshot = {} }

      catalog = {
        name           = "mydb"
        default_action = "Include"
        schemas = {
          public = {
            tables = {
              audit_log    = { action = "Exclude" }
              debug_events = { action = "Exclude" }
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

# Pattern 2. Replicate only specific tables.
# Set schema and default to Exclude, then Include individual tables.
resource "supermetal_connector" "exclude_by_default" {
  id   = "exclude-by-default"
  name = "Exclude by default"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Disable"

      replication_type = { snapshot = {} }

      catalog = {
        name           = "mydb"
        default_action = "Exclude"
        schemas = {
          public = {
            action = "Exclude"
            tables = {
              orders      = { action = "Include" }
              order_items = { action = "Include" }
              customers   = { action = "Include" }
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

# Pattern 3. Replicate entire schemas
resource "supermetal_connector" "full_schema" {
  id   = "full-schema"
  name = "Full schema"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Disable"

      replication_type = { snapshot = {} }

      catalog = {
        name           = "mydb"
        default_action = "Exclude"
        schemas = {
          analytics = { action = "Include", tables = {} }
          reporting = { action = "Include", tables = {} }
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
