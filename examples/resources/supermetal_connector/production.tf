# Production setup with logical replication, SSL, table selection,
# and lifecycle protection against accidental deletion.

resource "supermetal_connector" "production" {
  id   = "orders-cdc"
  name = "Orders CDC pipeline"

  source = {
    postgres = {
      host          = "postgres.internal"
      port          = 5432
      database      = "production"
      user          = "replicator"
      password      = var.pg_password
      ssl_mode      = "VerifyFull"
      ssl_root_cert = var.pg_ssl_root_cert  # PEM encoded CA certificate

      replication_type = {
        logical_replication = {
          publication_name = "supermetal_orders"
        }
      }

      catalog = {
        name           = "production"
        default_action = "Exclude"
        schemas = {
          public = {
            tables = {
              orders      = {}
              order_items = {}
              customers   = {}
            }
          }
        }
      }
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
          private_key_password = var.snowflake_key_password
        }
      }
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}
