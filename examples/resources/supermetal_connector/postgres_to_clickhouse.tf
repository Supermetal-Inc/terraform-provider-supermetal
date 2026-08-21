# Snapshot PostgreSQL into ClickHouse. Atomic swap loads each snapshot into a
# replacement table and swaps it into place after the load completes.

variable "clickhouse_url" {
  type = string
}

variable "clickhouse_user" {
  type = string
}

variable "clickhouse_password" {
  type      = string
  sensitive = true
}

resource "supermetal_connector" "postgres_to_clickhouse" {
  id   = "orders-to-clickhouse"
  name = "PostgreSQL to ClickHouse"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Require"

      replication_type = {
        snapshot = {}
      }
    }
  }

  sink = {
    clickhouse = {
      http_url           = var.clickhouse_url
      user               = var.clickhouse_user
      password           = var.clickhouse_password
      target_database    = "analytics"
      snapshot_load_mode = "StagedRename"
    }
  }
}
