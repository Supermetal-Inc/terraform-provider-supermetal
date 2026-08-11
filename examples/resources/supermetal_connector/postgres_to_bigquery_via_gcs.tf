# PostgreSQL to BigQuery in merge mode. BigQuery merge writes Parquet files to
# the GCS buffer before loading and merging them into destination tables.

variable "bigquery_service_account_key" {
  type      = string
  sensitive = true
}

variable "postgres_tables" {
  description = "PostgreSQL tables to replicate from the public schema"
  type        = set(string)
}

resource "supermetal_connector" "postgres_to_bigquery_via_gcs" {
  id   = "orders-to-bigquery"
  name = "PostgreSQL to BigQuery via GCS"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Require"

      replication_type = {
        logical_replication = {}
      }

      catalog = {
        name           = var.pg_database
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              for table_name in var.postgres_tables : table_name => {}
            }
          }
        }
      }
    }
  }

  sink = {
    big_query = {
      project_id = "analytics-project"
      dataset    = "raw"

      auth = {
        service_account_key = {
          key_json = var.bigquery_service_account_key
        }
      }

      write_mode = {
        merge = {}
      }
    }
  }

  buffer = {
    object_store = {
      url = "gs://supermetal-staging/orders"
      options = {
        service_account_key = {
          value = var.bigquery_service_account_key
        }
      }
    }
  }
}
