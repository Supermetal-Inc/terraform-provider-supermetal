# DB2 source with a DuckDB destination.

variable "db2_password" {
  type      = string
  sensitive = true
}

resource "supermetal_connector" "db2_to_duckdb" {
  id   = "db2-to-duckdb"
  name = "DB2 to DuckDB"

  source = {
    db2 = {
      host     = "db2.internal"
      port     = 50000
      database = "PRODUCTION"
      user     = "db2inst1"
      password = var.db2_password

      replication_type = {
        snapshot = {}
      }
    }
  }

  sink = {
    duckdb = {
      target_database = "main"
      connection = {
        quack = {
          url = var.duckdb_url
        }
      }
    }
  }
}
