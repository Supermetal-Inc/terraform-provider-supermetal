# Minimal quickstart. Snapshot all tables from PostgreSQL into DuckDB.

resource "supermetal_connector" "quickstart" {
  id   = "quickstart"
  name = "Quickstart example"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Disable"

      replication_type = {
        snapshot = {}
      }
    }
  }

  sink = {
    duckdb = {
      target_database = "analytics"
      connection = {
        quack = { url = var.duckdb_url }
      }
    }
  }
}
