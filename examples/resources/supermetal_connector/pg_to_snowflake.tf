# Snowflake sink with two authentication methods

variable "gcs_service_account_key" {
  type      = string
  sensitive = true
}

# Password authentication with buffered CDC and scheduled merges
resource "supermetal_connector" "snowflake_password" {
  id   = "to-snowflake-password"
  name = "Snowflake (password auth)"

  source = {
    postgres = {
      host     = var.pg_host
      port     = 5432
      database = var.pg_database
      user     = var.pg_user
      password = var.pg_password
      ssl_mode = "Disable"

      replication_type = { logical_replication = {} }
    }
  }

  sink = {
    snowflake = {
      account_identifier = var.snowflake_account  # e.g., "xy12345.us-east-1"
      user               = var.snowflake_user
      warehouse          = "COMPUTE_WH"
      target_database    = "ANALYTICS"
      target_schema      = "RAW"

      auth = {
        password = {
          password = var.snowflake_password
        }
      }

      merge_schedule = {
        peak_window = {
          starts_at = {
            expression = "0 8 * * 1-5"
            timezone   = "America/Los_Angeles"
          }
          ends_at = {
            expression = "0 18 * * 1-5"
            timezone   = "America/Los_Angeles"
          }
        }
        peak_flush_interval_ms     = 60000
        off_peak_flush_interval_ms = 21600000
      }
    }
  }

  buffer = {
    object_store = {
      url = "gs://supermetal-staging/snowflake"
      options = {
        service_account_key = {
          value = var.gcs_service_account_key
        }
      }
    }
  }
}

# Key pair authentication (recommended for production)
resource "supermetal_connector" "snowflake_keypair" {
  id   = "to-snowflake-keypair"
  name = "Snowflake (key pair auth)"

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
    snowflake = {
      account_identifier = var.snowflake_account
      user               = var.snowflake_user
      warehouse          = "COMPUTE_WH"
      target_database    = "ANALYTICS"
      target_schema      = "RAW"

      auth = {
        key_pair = {
          private_key_pem      = var.snowflake_private_key
          private_key_password = var.snowflake_key_password  # omit if unencrypted
        }
      }
    }
  }
}
