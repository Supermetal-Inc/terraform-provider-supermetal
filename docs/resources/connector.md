---
page_title: "supermetal_connector Resource"
description: "Manages a Supermetal CDC connector."
---

# supermetal_connector (Resource)

Manages a [Supermetal CDC connector](https://docs.supermetal.io/docs/main/). A connector
replicates data from exactly one source to exactly one target.

For source and target setup guides, see the
[Supermetal documentation](https://docs.supermetal.io/docs/main/).

## Example Usage

### PostgreSQL to DuckDB

```terraform
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
```

### PostgreSQL to Snowflake

```terraform
# Snowflake sink with two authentication methods

# Password authentication
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

      replication_type = { snapshot = {} }
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
```

### Production setup

```terraform
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
```

### SSH tunnel

```terraform
# SSH tunnel for databases behind a bastion host.
# The database host and port are as seen from the bastion, not your network.

resource "supermetal_connector" "tunneled" {
  id   = "tunneled-connector"
  name = "Database via SSH tunnel"

  source = {
    postgres = {
      # Host as seen from the bastion (private IP or localhost)
      host     = "10.0.1.50"
      port     = 5432
      database = "app"
      user     = "replicator"
      password = var.pg_password
      ssl_mode = "Disable"

      tunnel = {
        ssh = {
          bastion_host = "bastion.example.com"
          bastion_port = 22
          user         = "tunnel"
          auth = {
            bring_your_own_key = {
              private_key = var.ssh_private_key
            }
          }
        }
      }

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

# To generate a key pair for the tunnel, use tls_private_key
# and pass private_key_openssh to the bring_your_own_key block.
#
#   resource "tls_private_key" "tunnel" {
#     algorithm = "ED25519"
#   }
#
#   output "tunnel_public_key" {
#     value       = tls_private_key.tunnel.public_key_openssh
#     description = "Add to ~/.ssh/authorized_keys on the bastion"
#   }
```

### Table selection

```terraform
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
# Tables listed under an Exclude default are implicitly included.
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
```

### Disabled connector

```terraform
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
```

### Shared connection

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) Unique identifier for this connector. Used in the API path and for import. Must contain only letters, numbers, hyphens, and underscores (max 30 characters).
- `sink` (Attributes) Sink configuration. Exactly one sink type must be specified. (see [below for nested schema](#nestedatt--sink))
- `source` (Attributes) Source configuration. Exactly one source type must be specified. (see [below for nested schema](#nestedatt--source))

### Optional

- `disabled` (Boolean) Whether this connector is disabled.
- `name` (String) Display name for the connector.

<a id="nestedatt--sink"></a>
### Nested Schema for `sink`

Optional:

- `big_query` (Attributes) BigQuery destination (see [below for nested schema](#nestedatt--sink--big_query))
- `clickhouse` (Attributes) Configuration for a ClickHouse data sink (see [below for nested schema](#nestedatt--sink--clickhouse))
- `databricks` (Attributes) Databricks destination (see [below for nested schema](#nestedatt--sink--databricks))
- `doris` (Attributes) Apache Doris sink configuration (see [below for nested schema](#nestedatt--sink--doris))
- `duckdb` (Attributes) DuckDB sink (see [below for nested schema](#nestedatt--sink--duckdb))
- `iceberg` (Attributes) Iceberg destination (see [below for nested schema](#nestedatt--sink--iceberg))
- `kafka` (Attributes) Kafka destination (see [below for nested schema](#nestedatt--sink--kafka))
- `motherduck` (Attributes) DuckDB sink (see [below for nested schema](#nestedatt--sink--motherduck))
- `postgres` (Attributes) PostgreSQL destination (see [below for nested schema](#nestedatt--sink--postgres))
- `snowflake` (Attributes) Snowflake destination (see [below for nested schema](#nestedatt--sink--snowflake))
- `webhook` (Attributes) Webhook destination (see [below for nested schema](#nestedatt--sink--webhook))

<a id="nestedatt--sink--big_query"></a>
### Nested Schema for `sink.big_query`

Required:

- `auth` (Attributes) Authentication method for connecting to BigQuery (see [below for nested schema](#nestedatt--sink--big_query--auth))
- `dataset` (String)
- `project_id` (String)
- `write_mode` (Attributes) How the target writes data into BigQuery (see [below for nested schema](#nestedatt--sink--big_query--write_mode))

Optional:

- `disable_schema_prefix` (Boolean)
- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--big_query--history_mode))
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--big_query--migration_strategy))
- `query_priority` (String) Priority for BigQuery query jobs such as DDL, MERGE, and scripts. Load jobs are unaffected.

<a id="nestedatt--sink--big_query--auth"></a>
### Nested Schema for `sink.big_query.auth`

Optional:

- `service_account_key` (Attributes) Service account JSON key (see [below for nested schema](#nestedatt--sink--big_query--auth--service_account_key))

<a id="nestedatt--sink--big_query--auth--service_account_key"></a>
### Nested Schema for `sink.big_query.auth.service_account_key`

Required:

- `key_json` (String, Sensitive) Service account JSON key contents (the full JSON document inline)



<a id="nestedatt--sink--big_query--write_mode"></a>
### Nested Schema for `sink.big_query.write_mode`

Optional:

- `merge` (Attributes) Load Parquet into a staging table, then MERGE into the target (see [below for nested schema](#nestedatt--sink--big_query--write_mode--merge))
- `storage_write_api` (Attributes) Stream changes row by row via the BigQuery Storage Write API (see [below for nested schema](#nestedatt--sink--big_query--write_mode--storage_write_api))

<a id="nestedatt--sink--big_query--write_mode--merge"></a>
### Nested Schema for `sink.big_query.write_mode.merge`

Optional:

- `use_transactions` (Boolean) Wrap DML across multiple tables in a BigQuery script transaction for atomicity


<a id="nestedatt--sink--big_query--write_mode--storage_write_api"></a>
### Nested Schema for `sink.big_query.write_mode.storage_write_api`

Optional:

- `max_staleness_minutes` (Number) Maximum time readers may see stale data, in minutes. Sets BigQuery's background apply window.



<a id="nestedatt--sink--big_query--history_mode"></a>
### Nested Schema for `sink.big_query.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--big_query--history_mode--append))

<a id="nestedatt--sink--big_query--history_mode--append"></a>
### Nested Schema for `sink.big_query.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--big_query--migration_strategy"></a>
### Nested Schema for `sink.big_query.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)



<a id="nestedatt--sink--clickhouse"></a>
### Nested Schema for `sink.clickhouse`

Required:

- `http_url` (String)
- `target_database` (String)

Optional:

- `async_inserts` (Boolean)
- `disable_schema_prefix` (Boolean)
- `engine` (String) ClickHouse table engine
- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--clickhouse--history_mode))
- `max_snapshot_concurrency` (Number)
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--clickhouse--migration_strategy))
- `non_nullable_columns` (Boolean)
- `password` (String, Sensitive)
- `preserve_source_nullability` (Boolean)
- `ssl_client_cert_pem` (String, Sensitive)
- `ssl_client_key_pem` (String, Sensitive)
- `ssl_root_cert` (String, Sensitive)
- `ssl_verify` (Boolean)
- `table_name_modifier` (Attributes) Modifier for target table names. (see [below for nested schema](#nestedatt--sink--clickhouse--table_name_modifier))
- `ttl_days` (Number)
- `user` (String)

<a id="nestedatt--sink--clickhouse--history_mode"></a>
### Nested Schema for `sink.clickhouse.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--clickhouse--history_mode--append))

<a id="nestedatt--sink--clickhouse--history_mode--append"></a>
### Nested Schema for `sink.clickhouse.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--clickhouse--migration_strategy"></a>
### Nested Schema for `sink.clickhouse.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)


<a id="nestedatt--sink--clickhouse--table_name_modifier"></a>
### Nested Schema for `sink.clickhouse.table_name_modifier`

Optional:

- `prefix` (Attributes) Custom prefix prepended to table name (e.g. "raw_") (see [below for nested schema](#nestedatt--sink--clickhouse--table_name_modifier--prefix))
- `suffix` (Attributes) Custom suffix appended to table name (e.g. "_log") (see [below for nested schema](#nestedatt--sink--clickhouse--table_name_modifier--suffix))

<a id="nestedatt--sink--clickhouse--table_name_modifier--prefix"></a>
### Nested Schema for `sink.clickhouse.table_name_modifier.prefix`

Required:

- `value` (String)


<a id="nestedatt--sink--clickhouse--table_name_modifier--suffix"></a>
### Nested Schema for `sink.clickhouse.table_name_modifier.suffix`

Required:

- `value` (String)




<a id="nestedatt--sink--databricks"></a>
### Nested Schema for `sink.databricks`

Required:

- `auth` (Attributes) Authentication method for connecting to Databricks (see [below for nested schema](#nestedatt--sink--databricks--auth))
- `target_catalog` (String)
- `volume` (String)
- `warehouse` (String)

Optional:

- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--databricks--history_mode))
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--databricks--migration_strategy))
- `storage_credential` (String)
- `table_features` (Attributes) Delta table features to enable. If false, the workspace default is respected and Supermetal does not explicitly disable features. (see [below for nested schema](#nestedatt--sink--databricks--table_features))
- `target_schema` (String)

<a id="nestedatt--sink--databricks--auth"></a>
### Nested Schema for `sink.databricks.auth`

Optional:

- `cli` (Attributes) Databricks CLI authentication (see [below for nested schema](#nestedatt--sink--databricks--auth--cli))
- `m2m` (Attributes) OAuth M2M Service Principal authentication (see [below for nested schema](#nestedatt--sink--databricks--auth--m2m))
- `pat` (Attributes) Personal Access Token (PAT) authentication (see [below for nested schema](#nestedatt--sink--databricks--auth--pat))

<a id="nestedatt--sink--databricks--auth--cli"></a>
### Nested Schema for `sink.databricks.auth.cli`


<a id="nestedatt--sink--databricks--auth--m2m"></a>
### Nested Schema for `sink.databricks.auth.m2m`

Required:

- `client_id` (String) Client ID of the Service Principal
- `client_secret` (String, Sensitive) Client Secret of the Service Principal
- `workspace_host` (String) Databricks workspace hostname ("your-workspace.cloud.databricks.com")


<a id="nestedatt--sink--databricks--auth--pat"></a>
### Nested Schema for `sink.databricks.auth.pat`

Required:

- `personal_access_token` (String, Sensitive) Databricks Personal Access Token value ("dapi...")
- `workspace_host` (String) Databricks workspace hostname ("your-workspace.cloud.databricks.com")



<a id="nestedatt--sink--databricks--history_mode"></a>
### Nested Schema for `sink.databricks.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--databricks--history_mode--append))

<a id="nestedatt--sink--databricks--history_mode--append"></a>
### Nested Schema for `sink.databricks.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--databricks--migration_strategy"></a>
### Nested Schema for `sink.databricks.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)


<a id="nestedatt--sink--databricks--table_features"></a>
### Nested Schema for `sink.databricks.table_features`

Optional:

- `deletion_vectors` (Boolean) Enable deletion vectors. Note that Databricks deletion vectors are not compatible with Delta UniForm.
- `delta_uniform` (Boolean) Enables Delta UniForm support. This also enables column mapping and upgrades the minimum Databricks protocol to reader version 2 or writer version 5. See https://docs.delta.io/latest/delta-column-mapping.html
- `type_widening` (Boolean) Enable Delta Type Widening, which promotes columns to larger integral types



<a id="nestedatt--sink--doris"></a>
### Nested Schema for `sink.doris`

Required:

- `fe_http_url` (String)
- `target_database` (String)
- `user` (String)

Optional:

- `binary_handling_mode` (String) Encoding for binary columns stored as Apache Doris STRING
- `disable_schema_prefix` (Boolean)
- `fe_mysql_pool_max` (Number)
- `fe_mysql_port` (Number)
- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--doris--history_mode))
- `max_snapshot_concurrency` (Number)
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--doris--migration_strategy))
- `password` (String, Sensitive)
- `preserve_source_nullability` (Boolean)
- `ssl_client_cert_pem` (String, Sensitive)
- `ssl_client_key_pem` (String, Sensitive)
- `ssl_root_cert` (String, Sensitive)
- `ssl_verify` (Boolean)
- `table_model` (String) Apache Doris table model (Unique Key, Duplicate Key, or Auto).
- `table_name_modifier` (Attributes) Optional prefix or suffix on target table names (see [below for nested schema](#nestedatt--sink--doris--table_name_modifier))

<a id="nestedatt--sink--doris--history_mode"></a>
### Nested Schema for `sink.doris.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--doris--history_mode--append))

<a id="nestedatt--sink--doris--history_mode--append"></a>
### Nested Schema for `sink.doris.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--doris--migration_strategy"></a>
### Nested Schema for `sink.doris.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)


<a id="nestedatt--sink--doris--table_name_modifier"></a>
### Nested Schema for `sink.doris.table_name_modifier`

Optional:

- `prefix` (Attributes) Prefix added before the table name, for example "raw_". (see [below for nested schema](#nestedatt--sink--doris--table_name_modifier--prefix))
- `suffix` (Attributes) Suffix added after the table name, for example "_log". (see [below for nested schema](#nestedatt--sink--doris--table_name_modifier--suffix))

<a id="nestedatt--sink--doris--table_name_modifier--prefix"></a>
### Nested Schema for `sink.doris.table_name_modifier.prefix`

Required:

- `value` (String)


<a id="nestedatt--sink--doris--table_name_modifier--suffix"></a>
### Nested Schema for `sink.doris.table_name_modifier.suffix`

Required:

- `value` (String)




<a id="nestedatt--sink--duckdb"></a>
### Nested Schema for `sink.duckdb`

Required:

- `connection` (Attributes) Connection protocol (see [below for nested schema](#nestedatt--sink--duckdb--connection))
- `target_database` (String)

Optional:

- `enable_primary_keys` (Boolean)
- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--duckdb--history_mode))
- `max_snapshot_concurrency` (Number)
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--duckdb--migration_strategy))
- `preserve_source_nullability` (Boolean)
- `target_schema` (String)
- `use_transactions` (Boolean)

<a id="nestedatt--sink--duckdb--connection"></a>
### Nested Schema for `sink.duckdb.connection`

Optional:

- `pg` (Attributes) MotherDuck Postgres endpoint (https://motherduck.com/docs/sql-reference/postgres-endpoint/) (see [below for nested schema](#nestedatt--sink--duckdb--connection--pg))
- `quack` (Attributes) Quack binary protocol over HTTP (https://duckdb.org/quack/) (see [below for nested schema](#nestedatt--sink--duckdb--connection--quack))

<a id="nestedatt--sink--duckdb--connection--pg"></a>
### Nested Schema for `sink.duckdb.connection.pg`

Required:

- `database` (String) Name of the database to connect to
- `host` (String) Database server hostname or IP address ("localhost" or "mydb.123456789012.us-east-1.rds.amazonaws.com")
- `password` (String, Sensitive) Password for database authentication
- `user` (String) Username for database authentication

Optional:

- `max_pool_size` (Number) Maximum number of connections in the connection pool (0 for default)
- `operation_lock_timeout_seconds` (Number) Enables fail-fast behavior for data operations (COPY, INSERT, MERGE) by setting a `lock_timeout`. This prevents operations from waiting indefinitely when tables are locked by either long running transactions, DDL or Maintenance operations. Disabled by default, operations wait indefinitely. Set to a non-zero value (e.g., '60') to let operations fail-fast. https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-LOCK-TIMEOUT
- `port` (Number) Port number for the PostgreSQL server
- `ssl_cert` (String, Sensitive) Client's SSL certificate content
- `ssl_key` (String, Sensitive) Client's private SSL key content
- `ssl_mode` (String)
- `ssl_root_cert` (String, Sensitive) SSL root certificate content for server verification
- `tunnel` (Attributes) (see [below for nested schema](#nestedatt--sink--duckdb--connection--pg--tunnel))

<a id="nestedatt--sink--duckdb--connection--pg--tunnel"></a>
### Nested Schema for `sink.duckdb.connection.pg.tunnel`

Optional:

- `ssh` (Attributes) Tunnel through an SSH bastion host (see [below for nested schema](#nestedatt--sink--duckdb--connection--pg--tunnel--ssh))

<a id="nestedatt--sink--duckdb--connection--pg--tunnel--ssh"></a>
### Nested Schema for `sink.duckdb.connection.pg.tunnel.ssh`

Required:

- `auth` (Attributes) How to authenticate against the bastion (see [below for nested schema](#nestedatt--sink--duckdb--connection--pg--tunnel--ssh--auth))
- `bastion_host` (String) Hostname or IP of the SSH bastion server
- `user` (String) SSH username on the bastion server

Optional:

- `bastion_alternates` (List of String) Fallback bastion hostnames, tried in order if the primary is unreachable
- `bastion_port` (Number) SSH port on the bastion server

<a id="nestedatt--sink--duckdb--connection--pg--tunnel--ssh--auth"></a>
### Nested Schema for `sink.duckdb.connection.pg.tunnel.ssh.auth`

Optional:

- `bring_your_own_key` (Attributes) Paste your own private key (see [below for nested schema](#nestedatt--sink--duckdb--connection--pg--tunnel--ssh--auth--bring_your_own_key))
- `generated_key` (Attributes) Supermetal generates the keypair; you install the public key on the bastion (see [below for nested schema](#nestedatt--sink--duckdb--connection--pg--tunnel--ssh--auth--generated_key))

<a id="nestedatt--sink--duckdb--connection--pg--tunnel--ssh--auth--bring_your_own_key"></a>
### Nested Schema for `sink.duckdb.connection.pg.tunnel.ssh.auth.bring_your_own_key`

Required:

- `private_key` (String, Sensitive) OpenSSH-encoded private key


<a id="nestedatt--sink--duckdb--connection--pg--tunnel--ssh--auth--generated_key"></a>
### Nested Schema for `sink.duckdb.connection.pg.tunnel.ssh.auth.generated_key`

Required:

- `private_key` (String, Sensitive) Private key (managed by Supermetal)
- `public_key` (String) Public key — add this line to ~/.ssh/authorized_keys on your bastion






<a id="nestedatt--sink--duckdb--connection--quack"></a>
### Nested Schema for `sink.duckdb.connection.quack`

Required:

- `url` (String) URL of the Quack endpoint (e.g. "http://localhost:9494")

Optional:

- `ssl` (Boolean) Verify the server's SSL certificate
- `token` (String, Sensitive) Authentication token



<a id="nestedatt--sink--duckdb--history_mode"></a>
### Nested Schema for `sink.duckdb.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--duckdb--history_mode--append))

<a id="nestedatt--sink--duckdb--history_mode--append"></a>
### Nested Schema for `sink.duckdb.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--duckdb--migration_strategy"></a>
### Nested Schema for `sink.duckdb.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)



<a id="nestedatt--sink--iceberg"></a>
### Nested Schema for `sink.iceberg`

Required:

- `catalog` (Attributes) Iceberg catalog type (see [below for nested schema](#nestedatt--sink--iceberg--catalog))
- `target_namespace` (List of String)

Optional:

- `max_catalog_concurrency` (Number)
- `metadata_compression` (String) Iceberg metadata file compression
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--iceberg--migration_strategy))
- `parquet` (Attributes) Parquet writer settings (see [below for nested schema](#nestedatt--sink--iceberg--parquet))
- `spec_version` (String) Iceberg table format version
- `storage_credentials` (Attributes) Storage credentials for accessing data files (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials))
- `truncate_table_if_exists` (Boolean)
- `type_conversion` (Attributes) Opt-in type conversions applied before writing to the target.
Omit a subfield to keep the target's default behaviour for that type. (see [below for nested schema](#nestedatt--sink--iceberg--type_conversion))
- `vended_credentials` (Boolean)
- `write_mode` (Attributes) Write mode for CDC operations (see [below for nested schema](#nestedatt--sink--iceberg--write_mode))

<a id="nestedatt--sink--iceberg--catalog"></a>
### Nested Schema for `sink.iceberg.catalog`

Optional:

- `glue` (Attributes) AWS Glue catalog (see [below for nested schema](#nestedatt--sink--iceberg--catalog--glue))
- `hms` (Attributes) Hive Metastore (see [below for nested schema](#nestedatt--sink--iceberg--catalog--hms))
- `rest` (Attributes) REST catalog (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest))
- `s3tables` (Attributes) AWS S3 Tables (see [below for nested schema](#nestedatt--sink--iceberg--catalog--s3tables))
- `snowflake` (Attributes) Snowflake Iceberg catalog (see [below for nested schema](#nestedatt--sink--iceberg--catalog--snowflake))
- `sql` (Attributes) SQL-based catalog (SQLite, PostgreSQL, MySQL) (see [below for nested schema](#nestedatt--sink--iceberg--catalog--sql))
- `unity` (Attributes) Databricks Unity Catalog (see [below for nested schema](#nestedatt--sink--iceberg--catalog--unity))

<a id="nestedatt--sink--iceberg--catalog--glue"></a>
### Nested Schema for `sink.iceberg.catalog.glue`

Required:

- `warehouse` (String) Warehouse location

Optional:

- `access_key_id` (String, Sensitive) AWS access key ID (alternative to profile)
- `catalog_id` (String) AWS catalog ID (optional, uses account default)
- `profile_name` (String) AWS profile name
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--glue--properties))
- `region` (String) AWS region
- `secret_access_key` (String, Sensitive) AWS secret access key
- `session_token` (String, Sensitive) AWS session token
- `uri` (String) Glue endpoint URI (optional, uses default AWS endpoint)

<a id="nestedatt--sink--iceberg--catalog--glue--properties"></a>
### Nested Schema for `sink.iceberg.catalog.glue.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--catalog--hms"></a>
### Nested Schema for `sink.iceberg.catalog.hms`

Required:

- `uri` (String) Thrift URI (e.g., "thrift://localhost:9083")
- `warehouse` (String) Warehouse location

Optional:

- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--hms--properties))
- `thrift_transport` (String) Thrift transport: "framed" or "buffered" (default: buffered)

<a id="nestedatt--sink--iceberg--catalog--hms--properties"></a>
### Nested Schema for `sink.iceberg.catalog.hms.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--catalog--rest"></a>
### Nested Schema for `sink.iceberg.catalog.rest`

Required:

- `uri` (String) Catalog endpoint (e.g., "http://localhost:8181")

Optional:

- `auth` (Attributes) (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest--auth))
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest--properties))
- `warehouse` (String) Warehouse identifier

<a id="nestedatt--sink--iceberg--catalog--rest--auth"></a>
### Nested Schema for `sink.iceberg.catalog.rest.auth`

Optional:

- `basic` (Attributes) Basic username/password (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest--auth--basic))
- `bearer` (Attributes) Static bearer token (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest--auth--bearer))
- `oauth2` (Attributes) OAuth2 client credentials (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest--auth--oauth2))
- `sigv4` (Attributes) AWS SigV4 signing (see [below for nested schema](#nestedatt--sink--iceberg--catalog--rest--auth--sigv4))

<a id="nestedatt--sink--iceberg--catalog--rest--auth--basic"></a>
### Nested Schema for `sink.iceberg.catalog.rest.auth.basic`

Required:

- `password` (String, Sensitive) Password
- `username` (String) Username


<a id="nestedatt--sink--iceberg--catalog--rest--auth--bearer"></a>
### Nested Schema for `sink.iceberg.catalog.rest.auth.bearer`

Required:

- `token` (String, Sensitive) Bearer token value


<a id="nestedatt--sink--iceberg--catalog--rest--auth--oauth2"></a>
### Nested Schema for `sink.iceberg.catalog.rest.auth.oauth2`

Required:

- `client_id` (String) Client ID
- `client_secret` (String, Sensitive) Client secret
- `token_endpoint` (String) OAuth2 token endpoint URL

Optional:

- `scope` (String) OAuth2 scope


<a id="nestedatt--sink--iceberg--catalog--rest--auth--sigv4"></a>
### Nested Schema for `sink.iceberg.catalog.rest.auth.sigv4`

Required:

- `region` (String) AWS region for signing requests

Optional:

- `access_key_id` (String) AWS access key ID
- `profile_name` (String) AWS profile name (alternative to explicit credentials)
- `secret_access_key` (String, Sensitive) AWS secret access key
- `service` (String) Service name for signing (e.g., "s3tables")
- `session_token` (String, Sensitive) AWS session token



<a id="nestedatt--sink--iceberg--catalog--rest--properties"></a>
### Nested Schema for `sink.iceberg.catalog.rest.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--catalog--s3tables"></a>
### Nested Schema for `sink.iceberg.catalog.s3tables`

Required:

- `table_bucket_arn` (String) S3 table bucket ARN

Optional:

- `access_key_id` (String) AWS access key ID
- `endpoint` (String) Custom endpoint
- `profile_name` (String) AWS profile name
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--s3tables--properties))
- `region` (String) AWS region
- `secret_access_key` (String, Sensitive) AWS secret access key
- `session_token` (String, Sensitive) AWS session token

<a id="nestedatt--sink--iceberg--catalog--s3tables--properties"></a>
### Nested Schema for `sink.iceberg.catalog.s3tables.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--catalog--snowflake"></a>
### Nested Schema for `sink.iceberg.catalog.snowflake`

Required:

- `uri` (String) Snowflake account URL
- `warehouse` (String) Snowflake warehouse

Optional:

- `database` (String) Snowflake database
- `password` (String, Sensitive) Snowflake password
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--snowflake--properties))
- `role` (String) Snowflake role
- `schema` (String) Snowflake schema
- `username` (String) Snowflake username

<a id="nestedatt--sink--iceberg--catalog--snowflake--properties"></a>
### Nested Schema for `sink.iceberg.catalog.snowflake.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--catalog--sql"></a>
### Nested Schema for `sink.iceberg.catalog.sql`

Required:

- `uri` (String) Database connection URI (e.g., "sqlite://catalog.db", "postgresql://...")
- `warehouse` (String) Warehouse location

Optional:

- `bind_style` (String) SQL bind style: "DollarNumeric" (Postgres) or "QMark" (SQLite/MySQL)
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--sql--properties))

<a id="nestedatt--sink--iceberg--catalog--sql--properties"></a>
### Nested Schema for `sink.iceberg.catalog.sql.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--catalog--unity"></a>
### Nested Schema for `sink.iceberg.catalog.unity`

Required:

- `catalog_name` (String) Catalog name
- `uri` (String) Unity Catalog endpoint

Optional:

- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--catalog--unity--properties))
- `token` (String, Sensitive) Bearer token for authentication

<a id="nestedatt--sink--iceberg--catalog--unity--properties"></a>
### Nested Schema for `sink.iceberg.catalog.unity.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value




<a id="nestedatt--sink--iceberg--migration_strategy"></a>
### Nested Schema for `sink.iceberg.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)


<a id="nestedatt--sink--iceberg--parquet"></a>
### Nested Schema for `sink.iceberg.parquet`

Optional:

- `compression` (String)
- `compression_level` (Number) Compression level (1-22 for Zstd, 0-9 for Gzip/Brotli)
- `target_file_size_mb` (Number) Target file size in MB
- `version` (String)


<a id="nestedatt--sink--iceberg--storage_credentials"></a>
### Nested Schema for `sink.iceberg.storage_credentials`

Optional:

- `azure` (Attributes) Azure Storage (Blob and ADLS Gen2) (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--azure))
- `custom` (Attributes) Custom storage credentials (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--custom))
- `gcs` (Attributes) Google Cloud Storage (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--gcs))
- `s3` (Attributes) AWS S3 (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--s3))

<a id="nestedatt--sink--iceberg--storage_credentials--azure"></a>
### Nested Schema for `sink.iceberg.storage_credentials.azure`

Required:

- `account_name` (String) Storage account name

Optional:

- `account_key` (String, Sensitive) Account key (for shared key auth)
- `client_id` (String) Client ID (for service principal auth)
- `client_secret` (String, Sensitive) Client secret (for service principal auth)
- `endpoint` (String) Custom endpoint
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--azure--properties))
- `tenant_id` (String) Tenant ID (for service principal auth)

<a id="nestedatt--sink--iceberg--storage_credentials--azure--properties"></a>
### Nested Schema for `sink.iceberg.storage_credentials.azure.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--storage_credentials--custom"></a>
### Nested Schema for `sink.iceberg.storage_credentials.custom`

Optional:

- `properties` (Attributes List) Free-form key/value properties forwarded to the storage backend (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--custom--properties))

<a id="nestedatt--sink--iceberg--storage_credentials--custom--properties"></a>
### Nested Schema for `sink.iceberg.storage_credentials.custom.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--storage_credentials--gcs"></a>
### Nested Schema for `sink.iceberg.storage_credentials.gcs`

Optional:

- `credentials_json` (String, Sensitive) Service account key JSON (base64-encoded)
- `project_id` (String) GCS project ID
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--gcs--properties))

<a id="nestedatt--sink--iceberg--storage_credentials--gcs--properties"></a>
### Nested Schema for `sink.iceberg.storage_credentials.gcs.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value



<a id="nestedatt--sink--iceberg--storage_credentials--s3"></a>
### Nested Schema for `sink.iceberg.storage_credentials.s3`

Optional:

- `access_key_id` (String) AWS access key ID
- `endpoint` (String) Custom S3 endpoint for S3-compatible storage (e.g., MinIO)
- `iam_role_arn` (String) IAM role ARN to assume via STS
- `path_style_access` (Boolean) Use path-style access (for MinIO, etc.)
- `profile_name` (String) AWS profile name
- `properties` (Attributes List) Additional properties (see [below for nested schema](#nestedatt--sink--iceberg--storage_credentials--s3--properties))
- `region` (String) AWS region
- `secret_access_key` (String, Sensitive) AWS secret access key
- `session_token` (String, Sensitive) AWS session token for temporary credentials

<a id="nestedatt--sink--iceberg--storage_credentials--s3--properties"></a>
### Nested Schema for `sink.iceberg.storage_credentials.s3.properties`

Required:

- `key` (String) Property key
- `value` (String) Property value




<a id="nestedatt--sink--iceberg--type_conversion"></a>
### Nested Schema for `sink.iceberg.type_conversion`

Optional:

- `nanosecond` (Attributes) (see [below for nested schema](#nestedatt--sink--iceberg--type_conversion--nanosecond))
- `numeric` (Attributes) (see [below for nested schema](#nestedatt--sink--iceberg--type_conversion--numeric))

<a id="nestedatt--sink--iceberg--type_conversion--nanosecond"></a>
### Nested Schema for `sink.iceberg.type_conversion.nanosecond`

Optional:

- `mode` (String)


<a id="nestedatt--sink--iceberg--type_conversion--numeric"></a>
### Nested Schema for `sink.iceberg.type_conversion.numeric`

Optional:

- `mode` (String)
- `precision` (Number) Target decimal precision, 1-38 (Decimal mode only)
- `scale` (Number) Target decimal scale, 0 through precision (Decimal mode only)



<a id="nestedatt--sink--iceberg--write_mode"></a>
### Nested Schema for `sink.iceberg.write_mode`

Optional:

- `append` (Attributes) Append-only writes (see [below for nested schema](#nestedatt--sink--iceberg--write_mode--append))
- `merge_on_read` (Attributes) Row-level deletes using equality delete files (see [below for nested schema](#nestedatt--sink--iceberg--write_mode--merge_on_read))

<a id="nestedatt--sink--iceberg--write_mode--append"></a>
### Nested Schema for `sink.iceberg.write_mode.append`


<a id="nestedatt--sink--iceberg--write_mode--merge_on_read"></a>
### Nested Schema for `sink.iceberg.write_mode.merge_on_read`

Optional:

- `delete_mode` (String)
- `history_mode` (Attributes) (see [below for nested schema](#nestedatt--sink--iceberg--write_mode--merge_on_read--history_mode))
- `use_positional_deletes_only` (Boolean) Emit positional deletes only (default: false). Set true for
 Snowflake/Databricks readers that reject equality deletes. Requires a
 primary key and requires `delete_mode = Hard`, since positional deletes
 reference physical row offsets and cannot tombstone a row in place for
 soft-delete semantics. V2 tables emit positional delete files, V3
 tables emit deletion vectors.

<a id="nestedatt--sink--iceberg--write_mode--merge_on_read--history_mode"></a>
### Nested Schema for `sink.iceberg.write_mode.merge_on_read.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--iceberg--write_mode--merge_on_read--history_mode--append))

<a id="nestedatt--sink--iceberg--write_mode--merge_on_read--history_mode--append"></a>
### Nested Schema for `sink.iceberg.write_mode.merge_on_read.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`






<a id="nestedatt--sink--kafka"></a>
### Nested Schema for `sink.kafka`

Required:

- `connection` (Attributes) Kafka connection details and client settings (see [below for nested schema](#nestedatt--sink--kafka--connection))
- `format` (Attributes) Message format for Kafka payloads (see [below for nested schema](#nestedatt--sink--kafka--format))
- `topic_options` (Attributes) Defaults and optional per-topic configuration for topic creation (see [below for nested schema](#nestedatt--sink--kafka--topic_options))

Optional:

- `name` (String)
- `topic_name_template` (String)

<a id="nestedatt--sink--kafka--connection"></a>
### Nested Schema for `sink.kafka.connection`

Required:

- `brokers` (List of String) List of Kafka broker bootstrap server addresses ("host1:port1,host2:port2")

Optional:

- `auth` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--connection--auth))
- `config` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--connection--config))

<a id="nestedatt--sink--kafka--connection--auth"></a>
### Nested Schema for `sink.kafka.connection.auth`

Optional:

- `mechanism` (String)
- `password` (String, Sensitive) Password for SASL authentication
- `protocol` (String)
- `ssl_ca_location` (String) Server-side filesystem path to the CA certificate file (must be readable by the Supermetal process)
- `ssl_certificate_location` (String) Server-side filesystem path to the client certificate file (must be readable by the Supermetal process)
- `ssl_key_location` (String) Server-side filesystem path to the client private key file (must be readable by the Supermetal process)
- `ssl_key_password` (String, Sensitive) Password for the client private key
- `username` (String) Username for SASL authentication


<a id="nestedatt--sink--kafka--connection--config"></a>
### Nested Schema for `sink.kafka.connection.config`

Optional:

- `acks` (String)
- `batch_size` (Number) Maximum size of a batch of messages to send in bytes
- `client_id` (String) Optional Kafka client.id to set on the producer
- `compression_type` (String)
- `enable_idempotence` (Boolean) Enable idempotent producer (enforced/normalized when transactions are enabled)
- `global_properties` (Attributes List) Additional global/base client properties (key/value) (see [below for nested schema](#nestedatt--sink--kafka--connection--config--global_properties))
- `linger_ms` (Number) Time in milliseconds to wait for messages to accumulate in the producer batch
- `message_max_bytes` (Number) Maximum Kafka protocol message size in bytes
- `message_send_max_retries` (Number) Maximum number of retries for sending a message if it fails
- `message_timeout_ms` (Number) Per-message timeout (message.timeout.ms)
- `partitioner` (String) Partitioner name (e.g., "murmur2_random", "consistent_random")
- `producer_pool_size` (Number) Maximum number of producers in the snapshot producer pool (0 for default based on CPU cores)
- `producer_preset` (String)
- `producer_properties` (Attributes List) Additional producer-specific properties (key/value) (see [below for nested schema](#nestedatt--sink--kafka--connection--config--producer_properties))
- `queue_buffering_max_kbytes` (Number) Producer queue maximum size in kilobytes
- `queue_buffering_max_messages` (Number) Producer queue maximum number of messages
- `retry_backoff_max_ms` (Number) Maximum backoff time in milliseconds for retrying failed message sends
- `retry_backoff_ms` (Number) Time in milliseconds to wait before retrying a failed message send
- `shutdown_flush_timeout_ms` (Number) Flush duration on drop/close (ms)
- `timeout_ms` (Number) Kafka client communication timeout in milliseconds
- `topic_metadata_refresh_interval_ms` (Number) Interval in milliseconds for refreshing topic metadata from the broker
- `transactions` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--connection--config--transactions))

<a id="nestedatt--sink--kafka--connection--config--global_properties"></a>
### Nested Schema for `sink.kafka.connection.config.global_properties`

Required:

- `key` (String) Property key
- `value` (String) Property value


<a id="nestedatt--sink--kafka--connection--config--producer_properties"></a>
### Nested Schema for `sink.kafka.connection.config.producer_properties`

Required:

- `key` (String) Property key
- `value` (String) Property value


<a id="nestedatt--sink--kafka--connection--config--transactions"></a>
### Nested Schema for `sink.kafka.connection.config.transactions`

Optional:

- `enabled` (Boolean) Enable transactions
- `op_timeout_ms` (Number) Timeout for transaction operations like commit and abort in milliseconds (client-side deadline)
- `timeout_ms` (Number) transaction.timeout.ms (broker-side expiration)
- `transactional_id` (String) Explicit transactional.id; if empty and enabled, derive from KafkaSink.name




<a id="nestedatt--sink--kafka--format"></a>
### Nested Schema for `sink.kafka.format`

Optional:

- `debezium` (Attributes) Debezium-compatible message format (see [below for nested schema](#nestedatt--sink--kafka--format--debezium))
- `supermetal` (Attributes) Supermetal native message format (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal))

<a id="nestedatt--sink--kafka--format--debezium"></a>
### Nested Schema for `sink.kafka.format.debezium`

Required:

- `format_config` (Attributes) Base format configuration shared by all formats (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config))

Optional:

- `data_types` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--data_types))
- `emit_tx_metadata` (Boolean) Emit Debezium-style transaction metadata control events. Maps to Debezium's `provide.transaction.metadata`.
- `headers` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--headers))
- `include_before` (Boolean) Include "before" field in update/delete payloads
- `skipped_operations` (String) Comma-separated list of operation types to skip emitting. Valid values: c (create/insert), u (update), d (delete), t (truncate), none (skip nothing). Maps to Debezium's `skipped.operations`.
- `tombstones_on_delete` (Boolean) Whether a delete event is followed by a tombstone event. Maps to Debezium's `tombstones.on.delete`.
- `tx_metadata_topic` (String) Name of the transaction metadata topic (joined with the connector/topic prefix). Maps to Debezium's `topic.transaction`.

<a id="nestedatt--sink--kafka--format--debezium--format_config"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config`

Required:

- `value_serde` (Attributes) Serialization format for the message payload (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde))

Optional:

- `key_serde` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro--schema_registry))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro--schema_registry"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--json"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--schema_registry))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--options"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--schema_registry"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf--schema_registry))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--debezium--format_config--value_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.value_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication





<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro--schema_registry))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro--schema_registry"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--json"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--schema_registry))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--options"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--schema_registry"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf--schema_registry))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--debezium--format_config--key_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.debezium.format_config.key_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication






<a id="nestedatt--sink--kafka--format--debezium--data_types"></a>
### Nested Schema for `sink.kafka.format.debezium.data_types`

Optional:

- `binary_handling_mode` (String)
- `decimal_handling_mode` (String)
- `time_precision_mode` (String)


<a id="nestedatt--sink--kafka--format--debezium--headers"></a>
### Nested Schema for `sink.kafka.format.debezium.headers`

Optional:

- `context` (Boolean) Include __debezium.context.* headers
- `db` (Boolean) Database name header
- `db_name` (String) Header name (default: "db")
- `op` (Boolean) Operation header (c, r, u, d)
- `op_name` (String) Header name (default: "op")
- `pk_update` (Boolean) Include __debezium.newkey/__debezium.oldkey headers
- `prefix` (String) Prefix for header names
- `schema` (Boolean) Schema name header
- `schema_name` (String) Header name (default: "schema")
- `source_name` (String) Header name (default: "source")
- `source_template` (String) Templated source header. Variables: {db}, {schema}, {table}
- `tbl` (Boolean) Table name header
- `tbl_name` (String) Header name (default: "table")



<a id="nestedatt--sink--kafka--format--supermetal"></a>
### Nested Schema for `sink.kafka.format.supermetal`

Required:

- `format_config` (Attributes) Base format configuration shared by all formats (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config))

Optional:

- `tombstones_on_delete` (Boolean) Whether a delete event is followed by a tombstone event

<a id="nestedatt--sink--kafka--format--supermetal--format_config"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config`

Required:

- `value_serde` (Attributes) Serialization format for the message payload (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde))

Optional:

- `key_serde` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro--schema_registry))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro--schema_registry"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--schema_registry))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--options"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--schema_registry"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf--schema_registry))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--supermetal--format_config--value_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.value_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication





<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro--schema_registry))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro--schema_registry"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--schema_registry))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--options"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--schema_registry"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf--schema_registry))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--kafka--format--supermetal--format_config--key_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.kafka.format.supermetal.format_config.key_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication








<a id="nestedatt--sink--kafka--topic_options"></a>
### Nested Schema for `sink.kafka.topic_options`

Optional:

- `config` (Attributes List) Optional topic-level configuration entries (see [below for nested schema](#nestedatt--sink--kafka--topic_options--config))
- `naming_mode` (String)
- `operation_timeout_ms` (Number) Broker-side operation timeout for CreateTopics in milliseconds (0 => use client defaults)
- `partitions` (Number) Default partition count when creating topics
- `replication_factor` (Number) Default replication factor when creating topics

<a id="nestedatt--sink--kafka--topic_options--config"></a>
### Nested Schema for `sink.kafka.topic_options.config`

Required:

- `key` (String) Property key
- `value` (String) Property value




<a id="nestedatt--sink--motherduck"></a>
### Nested Schema for `sink.motherduck`

Required:

- `connection` (Attributes) Connection protocol (see [below for nested schema](#nestedatt--sink--motherduck--connection))
- `target_database` (String)

Optional:

- `enable_primary_keys` (Boolean)
- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--motherduck--history_mode))
- `max_snapshot_concurrency` (Number)
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--motherduck--migration_strategy))
- `preserve_source_nullability` (Boolean)
- `target_schema` (String)
- `use_transactions` (Boolean)

<a id="nestedatt--sink--motherduck--connection"></a>
### Nested Schema for `sink.motherduck.connection`

Optional:

- `pg` (Attributes) MotherDuck Postgres endpoint (https://motherduck.com/docs/sql-reference/postgres-endpoint/) (see [below for nested schema](#nestedatt--sink--motherduck--connection--pg))
- `quack` (Attributes) Quack binary protocol over HTTP (https://duckdb.org/quack/) (see [below for nested schema](#nestedatt--sink--motherduck--connection--quack))

<a id="nestedatt--sink--motherduck--connection--pg"></a>
### Nested Schema for `sink.motherduck.connection.pg`

Required:

- `database` (String) Name of the database to connect to
- `host` (String) Database server hostname or IP address ("localhost" or "mydb.123456789012.us-east-1.rds.amazonaws.com")
- `password` (String, Sensitive) Password for database authentication
- `user` (String) Username for database authentication

Optional:

- `max_pool_size` (Number) Maximum number of connections in the connection pool (0 for default)
- `operation_lock_timeout_seconds` (Number) Enables fail-fast behavior for data operations (COPY, INSERT, MERGE) by setting a `lock_timeout`. This prevents operations from waiting indefinitely when tables are locked by either long running transactions, DDL or Maintenance operations. Disabled by default, operations wait indefinitely. Set to a non-zero value (e.g., '60') to let operations fail-fast. https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-LOCK-TIMEOUT
- `port` (Number) Port number for the PostgreSQL server
- `ssl_cert` (String, Sensitive) Client's SSL certificate content
- `ssl_key` (String, Sensitive) Client's private SSL key content
- `ssl_mode` (String)
- `ssl_root_cert` (String, Sensitive) SSL root certificate content for server verification
- `tunnel` (Attributes) (see [below for nested schema](#nestedatt--sink--motherduck--connection--pg--tunnel))

<a id="nestedatt--sink--motherduck--connection--pg--tunnel"></a>
### Nested Schema for `sink.motherduck.connection.pg.tunnel`

Optional:

- `ssh` (Attributes) Tunnel through an SSH bastion host (see [below for nested schema](#nestedatt--sink--motherduck--connection--pg--tunnel--ssh))

<a id="nestedatt--sink--motherduck--connection--pg--tunnel--ssh"></a>
### Nested Schema for `sink.motherduck.connection.pg.tunnel.ssh`

Required:

- `auth` (Attributes) How to authenticate against the bastion (see [below for nested schema](#nestedatt--sink--motherduck--connection--pg--tunnel--ssh--auth))
- `bastion_host` (String) Hostname or IP of the SSH bastion server
- `user` (String) SSH username on the bastion server

Optional:

- `bastion_alternates` (List of String) Fallback bastion hostnames, tried in order if the primary is unreachable
- `bastion_port` (Number) SSH port on the bastion server

<a id="nestedatt--sink--motherduck--connection--pg--tunnel--ssh--auth"></a>
### Nested Schema for `sink.motherduck.connection.pg.tunnel.ssh.auth`

Optional:

- `bring_your_own_key` (Attributes) Paste your own private key (see [below for nested schema](#nestedatt--sink--motherduck--connection--pg--tunnel--ssh--auth--bring_your_own_key))
- `generated_key` (Attributes) Supermetal generates the keypair; you install the public key on the bastion (see [below for nested schema](#nestedatt--sink--motherduck--connection--pg--tunnel--ssh--auth--generated_key))

<a id="nestedatt--sink--motherduck--connection--pg--tunnel--ssh--auth--bring_your_own_key"></a>
### Nested Schema for `sink.motherduck.connection.pg.tunnel.ssh.auth.bring_your_own_key`

Required:

- `private_key` (String, Sensitive) OpenSSH-encoded private key


<a id="nestedatt--sink--motherduck--connection--pg--tunnel--ssh--auth--generated_key"></a>
### Nested Schema for `sink.motherduck.connection.pg.tunnel.ssh.auth.generated_key`

Required:

- `private_key` (String, Sensitive) Private key (managed by Supermetal)
- `public_key` (String) Public key — add this line to ~/.ssh/authorized_keys on your bastion






<a id="nestedatt--sink--motherduck--connection--quack"></a>
### Nested Schema for `sink.motherduck.connection.quack`

Required:

- `url` (String) URL of the Quack endpoint (e.g. "http://localhost:9494")

Optional:

- `ssl` (Boolean) Verify the server's SSL certificate
- `token` (String, Sensitive) Authentication token



<a id="nestedatt--sink--motherduck--history_mode"></a>
### Nested Schema for `sink.motherduck.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--motherduck--history_mode--append))

<a id="nestedatt--sink--motherduck--history_mode--append"></a>
### Nested Schema for `sink.motherduck.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--motherduck--migration_strategy"></a>
### Nested Schema for `sink.motherduck.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)



<a id="nestedatt--sink--postgres"></a>
### Nested Schema for `sink.postgres`

Required:

- `database` (String)
- `host` (String)
- `password` (String, Sensitive)
- `user` (String)

Optional:

- `max_pool_size` (Number)
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--postgres--migration_strategy))
- `operation_lock_timeout_seconds` (Number)
- `port` (Number)
- `ssl_cert` (String, Sensitive)
- `ssl_key` (String, Sensitive)
- `ssl_mode` (String) SSL connection mode for the PostgreSQL server
- `ssl_root_cert` (String, Sensitive)
- `target_schema` (String)
- `tunnel` (Attributes) Optional network transport. Leave unset for direct TCP; pick a
 variant to tunnel the connection. Today only SSH bastion is supported;
 additional transports (e.g. PrivateLink) can be added as new variants. (see [below for nested schema](#nestedatt--sink--postgres--tunnel))

<a id="nestedatt--sink--postgres--migration_strategy"></a>
### Nested Schema for `sink.postgres.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)


<a id="nestedatt--sink--postgres--tunnel"></a>
### Nested Schema for `sink.postgres.tunnel`

Optional:

- `ssh` (Attributes) Tunnel through an SSH bastion host (see [below for nested schema](#nestedatt--sink--postgres--tunnel--ssh))

<a id="nestedatt--sink--postgres--tunnel--ssh"></a>
### Nested Schema for `sink.postgres.tunnel.ssh`

Required:

- `auth` (Attributes) How to authenticate against the bastion (see [below for nested schema](#nestedatt--sink--postgres--tunnel--ssh--auth))
- `bastion_host` (String) Hostname or IP of the SSH bastion server
- `user` (String) SSH username on the bastion server

Optional:

- `bastion_alternates` (List of String) Fallback bastion hostnames, tried in order if the primary is unreachable
- `bastion_port` (Number) SSH port on the bastion server

<a id="nestedatt--sink--postgres--tunnel--ssh--auth"></a>
### Nested Schema for `sink.postgres.tunnel.ssh.auth`

Optional:

- `bring_your_own_key` (Attributes) Paste your own private key (see [below for nested schema](#nestedatt--sink--postgres--tunnel--ssh--auth--bring_your_own_key))
- `generated_key` (Attributes) Supermetal generates the keypair; you install the public key on the bastion (see [below for nested schema](#nestedatt--sink--postgres--tunnel--ssh--auth--generated_key))

<a id="nestedatt--sink--postgres--tunnel--ssh--auth--bring_your_own_key"></a>
### Nested Schema for `sink.postgres.tunnel.ssh.auth.bring_your_own_key`

Required:

- `private_key` (String, Sensitive) OpenSSH-encoded private key


<a id="nestedatt--sink--postgres--tunnel--ssh--auth--generated_key"></a>
### Nested Schema for `sink.postgres.tunnel.ssh.auth.generated_key`

Required:

- `private_key` (String, Sensitive) Private key (managed by Supermetal)
- `public_key` (String) Public key — add this line to ~/.ssh/authorized_keys on your bastion






<a id="nestedatt--sink--snowflake"></a>
### Nested Schema for `sink.snowflake`

Required:

- `account_identifier` (String)
- `auth` (Attributes) Authentication method for connecting to Snowflake (see [below for nested schema](#nestedatt--sink--snowflake--auth))
- `target_database` (String)
- `user` (String)
- `warehouse` (String)

Optional:

- `history_mode` (Attributes) How to preserve change history (see [below for nested schema](#nestedatt--sink--snowflake--history_mode))
- `migration_strategy` (Attributes) (see [below for nested schema](#nestedatt--sink--snowflake--migration_strategy))
- `role` (String)
- `target_schema` (String)
- `use_transactions` (Boolean)

<a id="nestedatt--sink--snowflake--auth"></a>
### Nested Schema for `sink.snowflake.auth`

Optional:

- `key_pair` (Attributes) Key pair authentication (see [below for nested schema](#nestedatt--sink--snowflake--auth--key_pair))
- `password` (Attributes) Password-based authentication (see [below for nested schema](#nestedatt--sink--snowflake--auth--password))

<a id="nestedatt--sink--snowflake--auth--key_pair"></a>
### Nested Schema for `sink.snowflake.auth.key_pair`

Required:

- `private_key_pem` (String, Sensitive) Private key content in PEM format ("-----BEGIN ENCRYPTED PRIVATE KEY-----")

Optional:

- `private_key_password` (String, Sensitive) Passphrase to decrypt the private key (if it's encrypted)


<a id="nestedatt--sink--snowflake--auth--password"></a>
### Nested Schema for `sink.snowflake.auth.password`

Required:

- `password` (String, Sensitive) User's password for authentication



<a id="nestedatt--sink--snowflake--history_mode"></a>
### Nested Schema for `sink.snowflake.history_mode`

Optional:

- `append` (Attributes) Write events to a parallel table (see [below for nested schema](#nestedatt--sink--snowflake--history_mode--append))

<a id="nestedatt--sink--snowflake--history_mode--append"></a>
### Nested Schema for `sink.snowflake.history_mode.append`

Optional:

- `suffix` (String) Suffix appended to the source table name to form the history table name, for example `_history` produces `orders_history`



<a id="nestedatt--sink--snowflake--migration_strategy"></a>
### Nested Schema for `sink.snowflake.migration_strategy`

Optional:

- `allowed` (String)
- `disable_all` (Boolean)



<a id="nestedatt--sink--webhook"></a>
### Nested Schema for `sink.webhook`

Required:

- `format` (Attributes) Message format for Kafka payloads (see [below for nested schema](#nestedatt--sink--webhook--format))
- `url` (String)

Optional:

- `auth` (Attributes) Authentication method (see [below for nested schema](#nestedatt--sink--webhook--auth))
- `batch` (Attributes) Batching behavior for outbound requests (see [below for nested schema](#nestedatt--sink--webhook--batch))
- `compression` (String) Request body compression
- `headers` (Attributes) Header configuration for outbound requests (see [below for nested schema](#nestedatt--sink--webhook--headers))
- `path_template` (String)
- `rate_limit` (Attributes) Rate limiting for outbound requests (see [below for nested schema](#nestedatt--sink--webhook--rate_limit))
- `request_timeout_ms` (Number)
- `retry` (Attributes) Retry behavior for failed requests (see [below for nested schema](#nestedatt--sink--webhook--retry))

<a id="nestedatt--sink--webhook--format"></a>
### Nested Schema for `sink.webhook.format`

Optional:

- `debezium` (Attributes) Debezium-compatible message format (see [below for nested schema](#nestedatt--sink--webhook--format--debezium))
- `supermetal` (Attributes) Supermetal native message format (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal))

<a id="nestedatt--sink--webhook--format--debezium"></a>
### Nested Schema for `sink.webhook.format.debezium`

Required:

- `format_config` (Attributes) Base format configuration shared by all formats (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config))

Optional:

- `data_types` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--data_types))
- `emit_tx_metadata` (Boolean) Emit Debezium-style transaction metadata control events. Maps to Debezium's `provide.transaction.metadata`.
- `headers` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--headers))
- `include_before` (Boolean) Include "before" field in update/delete payloads
- `skipped_operations` (String) Comma-separated list of operation types to skip emitting. Valid values: c (create/insert), u (update), d (delete), t (truncate), none (skip nothing). Maps to Debezium's `skipped.operations`.
- `tombstones_on_delete` (Boolean) Whether a delete event is followed by a tombstone event. Maps to Debezium's `tombstones.on.delete`.
- `tx_metadata_topic` (String) Name of the transaction metadata topic (joined with the connector/topic prefix). Maps to Debezium's `topic.transaction`.

<a id="nestedatt--sink--webhook--format--debezium--format_config"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config`

Required:

- `value_serde` (Attributes) Serialization format for the message payload (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde))

Optional:

- `key_serde` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro--schema_registry))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro--schema_registry"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--json"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--schema_registry))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--options"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--schema_registry"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf--schema_registry))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--debezium--format_config--value_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.value_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication





<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro--schema_registry))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro--schema_registry"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--json"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--schema_registry))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--options"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--schema_registry"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf--schema_registry))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--debezium--format_config--key_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.debezium.format_config.key_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication






<a id="nestedatt--sink--webhook--format--debezium--data_types"></a>
### Nested Schema for `sink.webhook.format.debezium.data_types`

Optional:

- `binary_handling_mode` (String)
- `decimal_handling_mode` (String)
- `time_precision_mode` (String)


<a id="nestedatt--sink--webhook--format--debezium--headers"></a>
### Nested Schema for `sink.webhook.format.debezium.headers`

Optional:

- `context` (Boolean) Include __debezium.context.* headers
- `db` (Boolean) Database name header
- `db_name` (String) Header name (default: "db")
- `op` (Boolean) Operation header (c, r, u, d)
- `op_name` (String) Header name (default: "op")
- `pk_update` (Boolean) Include __debezium.newkey/__debezium.oldkey headers
- `prefix` (String) Prefix for header names
- `schema` (Boolean) Schema name header
- `schema_name` (String) Header name (default: "schema")
- `source_name` (String) Header name (default: "source")
- `source_template` (String) Templated source header. Variables: {db}, {schema}, {table}
- `tbl` (Boolean) Table name header
- `tbl_name` (String) Header name (default: "table")



<a id="nestedatt--sink--webhook--format--supermetal"></a>
### Nested Schema for `sink.webhook.format.supermetal`

Required:

- `format_config` (Attributes) Base format configuration shared by all formats (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config))

Optional:

- `tombstones_on_delete` (Boolean) Whether a delete event is followed by a tombstone event

<a id="nestedatt--sink--webhook--format--supermetal--format_config"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config`

Required:

- `value_serde` (Attributes) Serialization format for the message payload (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde))

Optional:

- `key_serde` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro--schema_registry))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro--schema_registry"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--schema_registry))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--options"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--schema_registry"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf--schema_registry))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--supermetal--format_config--value_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.value_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication





<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde`

Optional:

- `avro` (Attributes) Avro message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro))
- `json` (Attributes) JSON message format (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json))
- `protobuf` (Attributes) Protobuf message format (requires Schema Registry) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.avro`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Avro serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro--schema_registry))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro--schema_registry"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.avro.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.avro.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--avro--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.avro.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.json`

Optional:

- `options` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--options))
- `schema_registry` (Attributes) (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--schema_registry))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--options"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.json.options`

Optional:

- `embed_schema_in_message` (Boolean) Include schema in message payloads (only applies if no schema registry is configured)


<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--schema_registry"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.json.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.json.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--json--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.json.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication




<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.protobuf`

Required:

- `schema_registry` (Attributes) Schema Registry configuration for Protobuf serialization/deserialization (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf--schema_registry))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf--schema_registry"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.protobuf.schema_registry`

Optional:

- `apicurio` (Attributes) Apicurio Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf--schema_registry--apicurio))
- `confluent` (Attributes) Confluent Schema Registry (see [below for nested schema](#nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf--schema_registry--confluent))

<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf--schema_registry--apicurio"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.protobuf.schema_registry.apicurio`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication


<a id="nestedatt--sink--webhook--format--supermetal--format_config--key_serde--protobuf--schema_registry--confluent"></a>
### Nested Schema for `sink.webhook.format.supermetal.format_config.key_serde.protobuf.schema_registry.confluent`

Required:

- `url` (String) URL of the Schema Registry ("http://localhost:8081")

Optional:

- `password` (String, Sensitive) Password for Schema Registry basic authentication
- `username` (String) Username for Schema Registry basic authentication








<a id="nestedatt--sink--webhook--auth"></a>
### Nested Schema for `sink.webhook.auth`

Optional:

- `basic` (Attributes) HTTP Basic authentication (see [below for nested schema](#nestedatt--sink--webhook--auth--basic))
- `bearer` (Attributes) Bearer token (see [below for nested schema](#nestedatt--sink--webhook--auth--bearer))
- `header` (Attributes) Custom header (e.g., X-API-Key) (see [below for nested schema](#nestedatt--sink--webhook--auth--header))

<a id="nestedatt--sink--webhook--auth--basic"></a>
### Nested Schema for `sink.webhook.auth.basic`

Required:

- `password` (String, Sensitive) Password
- `username` (String) Username


<a id="nestedatt--sink--webhook--auth--bearer"></a>
### Nested Schema for `sink.webhook.auth.bearer`

Required:

- `token` (String, Sensitive) Bearer token value


<a id="nestedatt--sink--webhook--auth--header"></a>
### Nested Schema for `sink.webhook.auth.header`

Required:

- `name` (String) Header name
- `value` (String, Sensitive) Header value (treated as a secret since it carries credentials)



<a id="nestedatt--sink--webhook--batch"></a>
### Nested Schema for `sink.webhook.batch`

Optional:

- `max_batch_size` (Number) Maximum messages per request (0 = unlimited)
- `max_payload_bytes` (Number) Maximum payload size in bytes per request


<a id="nestedatt--sink--webhook--headers"></a>
### Nested Schema for `sink.webhook.headers`

Optional:

- `idempotency_key` (Boolean) Include idempotency key header
- `idempotency_key_header` (String) Header name for idempotency key
- `source_schema` (Boolean) Include source schema header
- `source_schema_header` (String) Header name for source schema
- `source_table` (Boolean) Include source table header
- `source_table_header` (String) Header name for source table
- `static_` (Attributes Map) Additional static headers included in every request (see [below for nested schema](#nestedatt--sink--webhook--headers--static_))

<a id="nestedatt--sink--webhook--headers--static_"></a>
### Nested Schema for `sink.webhook.headers.static_`

Required:

- `value` (String) Header value



<a id="nestedatt--sink--webhook--rate_limit"></a>
### Nested Schema for `sink.webhook.rate_limit`

Optional:

- `max_bytes_per_second` (Number) Maximum bytes per second (0 = unlimited)
- `max_requests_per_second` (Number) Maximum requests per second (0 = unlimited)


<a id="nestedatt--sink--webhook--retry"></a>
### Nested Schema for `sink.webhook.retry`

Optional:

- `initial_backoff_ms` (Number) Initial delay between retries in milliseconds
- `max_backoff_ms` (Number) Maximum delay between retries in milliseconds
- `max_retries` (Number) Number of retry attempts (0 = unlimited)
- `max_retry_duration_ms` (Number) Maximum total retry duration in milliseconds (0 = unlimited)




<a id="nestedatt--source"></a>
### Nested Schema for `source`

Optional:

- `db2` (Attributes) Db2 replication source (see [below for nested schema](#nestedatt--source--db2))
- `file_source` (Attributes) Ingest files from object stores (S3, GCS, Azure Blob) or filesystems (local, SFTP) (see [below for nested schema](#nestedatt--source--file_source))
- `mongo` (Attributes) MongoDB replication source (see [below for nested schema](#nestedatt--source--mongo))
- `mysql` (Attributes) MySQL replication source (see [below for nested schema](#nestedatt--source--mysql))
- `oracle` (Attributes) Oracle replication source (see [below for nested schema](#nestedatt--source--oracle))
- `postgres` (Attributes) PostgreSQL replication source (see [below for nested schema](#nestedatt--source--postgres))
- `sqlserver` (Attributes) SQL Server replication source (see [below for nested schema](#nestedatt--source--sqlserver))

<a id="nestedatt--source--db2"></a>
### Nested Schema for `source.db2`

Required:

- `database` (String)
- `host` (String)
- `password` (String, Sensitive)
- `replication_type` (Attributes) Specifies the Db2 replication method (see [below for nested schema](#nestedatt--source--db2--replication_type))
- `user` (String)

Optional:

- `catalog` (Attributes) (see [below for nested schema](#nestedatt--source--db2--catalog))
- `keyless_table_strategy` (Attributes) How to replicate tables without a primary key (see [below for nested schema](#nestedatt--source--db2--keyless_table_strategy))
- `max_pool_size` (Number)
- `port` (Number)
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--db2--system_columns))

<a id="nestedatt--source--db2--replication_type"></a>
### Nested Schema for `source.db2.replication_type`

Optional:

- `snapshot` (Attributes) Initial snapshot / backfill only (see [below for nested schema](#nestedatt--source--db2--replication_type--snapshot))

<a id="nestedatt--source--db2--replication_type--snapshot"></a>
### Nested Schema for `source.db2.replication_type.snapshot`

Optional:

- `max_text_size` (Number) The maximum size we are allocating for a field in a column holding text. This protects
against the driver not knowing a sensible upper bound or The schema not being sanitized.

Increase this value, if snapshots fail due to truncated values. Defaults to 10 MiB.



<a id="nestedatt--source--db2--catalog"></a>
### Nested Schema for `source.db2.catalog`

Required:

- `name` (String)
- `schemas` (Attributes Map) (see [below for nested schema](#nestedatt--source--db2--catalog--schemas))

Optional:

- `default_action` (String)

<a id="nestedatt--source--db2--catalog--schemas"></a>
### Nested Schema for `source.db2.catalog.schemas`

Required:

- `tables` (Attributes Map) (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables))

Optional:

- `action` (String)

<a id="nestedatt--source--db2--catalog--schemas--tables"></a>
### Nested Schema for `source.db2.catalog.schemas.tables`

Optional:

- `action` (String)
- `columns` (Attributes Map) (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--columns))
- `iceberg_partition_spec` (Attributes) (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec))

<a id="nestedatt--source--db2--catalog--schemas--tables--columns"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.columns`

Optional:

- `action` (String)


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec`

Required:

- `fields` (Attributes List) Partition fields, applied in order (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields))

<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields`

Required:

- `source_column` (String) Source column name
- `transform` (Attributes) Transform applied to the source column (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform))

Optional:

- `name` (String) Partition column name in Iceberg. Defaults to {source_column}_{transform} (e.g. created_at_day)

<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform`

Optional:

- `bucket` (Attributes) Hash into a fixed number of buckets (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket))
- `day` (Attributes) Day of a date or timestamp (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day))
- `hour` (Attributes) Hour of a timestamp (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour))
- `identity` (Attributes) Source value, unchanged (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity))
- `month` (Attributes) Month of a date or timestamp (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month))
- `truncate` (Attributes) Truncate to a fixed width (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate))
- `year` (Attributes) Year of a date or timestamp (see [below for nested schema](#nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year))

<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.bucket`

Optional:

- `num_buckets` (Number) Number of hash buckets


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.day`


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.hour`


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.identity`


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.month`


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.truncate`

Optional:

- `width` (Number) Truncation width: characters for strings, bytes for binary, modulus for integers and decimals


<a id="nestedatt--source--db2--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year"></a>
### Nested Schema for `source.db2.catalog.schemas.tables.iceberg_partition_spec.fields.transform.year`








<a id="nestedatt--source--db2--keyless_table_strategy"></a>
### Nested Schema for `source.db2.keyless_table_strategy`

Optional:

- `append_only` (Attributes) (see [below for nested schema](#nestedatt--source--db2--keyless_table_strategy--append_only))
- `dedup_by_row_hash` (Attributes) (see [below for nested schema](#nestedatt--source--db2--keyless_table_strategy--dedup_by_row_hash))

<a id="nestedatt--source--db2--keyless_table_strategy--append_only"></a>
### Nested Schema for `source.db2.keyless_table_strategy.append_only`


<a id="nestedatt--source--db2--keyless_table_strategy--dedup_by_row_hash"></a>
### Nested Schema for `source.db2.keyless_table_strategy.dedup_by_row_hash`



<a id="nestedatt--source--db2--system_columns"></a>
### Nested Schema for `source.db2.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--db2--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--db2--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--db2--system_columns--synced_at))

<a id="nestedatt--source--db2--system_columns--lsn"></a>
### Nested Schema for `source.db2.system_columns.lsn`


<a id="nestedatt--source--db2--system_columns--op"></a>
### Nested Schema for `source.db2.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--db2--system_columns--synced_at"></a>
### Nested Schema for `source.db2.system_columns.synced_at`




<a id="nestedatt--source--file_source"></a>
### Nested Schema for `source.file_source`

Required:

- `object_store` (Attributes) Object store configuration (see [below for nested schema](#nestedatt--source--file_source--object_store))

Optional:

- `discovery` (Attributes) File discovery method (see [below for nested schema](#nestedatt--source--file_source--discovery))
- `error_handling` (Attributes) File processing error handling (see [below for nested schema](#nestedatt--source--file_source--error_handling))
- `exclude_patterns` (List of String)
- `format_details` (Attributes) Format-specific parsing options (see [below for nested schema](#nestedatt--source--file_source--format_details))
- `glob_patterns` (List of String)
- `post_processing` (Attributes) Action to take on source files after successful processing (see [below for nested schema](#nestedatt--source--file_source--post_processing))
- `start_date` (Number)
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--file_source--system_columns))
- `table_mapping` (Attributes) How files map to destination tables (see [below for nested schema](#nestedatt--source--file_source--table_mapping))

<a id="nestedatt--source--file_source--object_store"></a>
### Nested Schema for `source.file_source.object_store`

Required:

- `url` (String) URL: "s3://mybucket", "azure://mycontainer", "gs://mybucket", "file:///absolute/path"

Optional:

- `allow_http` (Boolean) Allow HTTP connections (default: false, HTTPS only)
- `allow_invalid_certificates` (Boolean) Allow invalid/self-signed certificates (default: false)
- `max_concurrent_parts` (Number) Max concurrent part uploads per file. Set to 1 for cross-region or to prevent part upload failures and timeouts due to limited bandwidth.
- `max_concurrent_requests` (Number) Max concurrent requests to the object store. 0 means no limit.
- `options` (Attributes Map) Configuration options (key-value pairs)

S3: [{"name": "region", "value": "us-east-1"}, {"name": "access_key_id", "value": "AKIA..."}, {"name": "secret_access_key", "value": "..."}]

Azure: [{"name": "account_name", "value": "myaccount"}, {"name": "access_key", "value": "..."} or {"name": "sas_token", "value": "sp=..."}]

GCS: [{"name": "service_account_key", "value": "{...JSON...}"}] (see [below for nested schema](#nestedatt--source--file_source--object_store--options))
- `root_certificate_pem` (String, Sensitive) PEM-encoded root certificate(s) for TLS verification

<a id="nestedatt--source--file_source--object_store--options"></a>
### Nested Schema for `source.file_source.object_store.options`

Required:

- `value` (String) Option value



<a id="nestedatt--source--file_source--discovery"></a>
### Nested Schema for `source.file_source.discovery`

Optional:

- `poll` (Attributes) Periodic polling for new and modified files (see [below for nested schema](#nestedatt--source--file_source--discovery--poll))

<a id="nestedatt--source--file_source--discovery--poll"></a>
### Nested Schema for `source.file_source.discovery.poll`

Optional:

- `interval_seconds` (Number) Seconds between file scans (0 for one-time sync)



<a id="nestedatt--source--file_source--error_handling"></a>
### Nested Schema for `source.file_source.error_handling`

Optional:

- `on_file_error` (String)


<a id="nestedatt--source--file_source--format_details"></a>
### Nested Schema for `source.file_source.format_details`

Optional:

- `csv` (Attributes) (see [below for nested schema](#nestedatt--source--file_source--format_details--csv))
- `json` (Attributes) (see [below for nested schema](#nestedatt--source--file_source--format_details--json))
- `parquet` (Attributes) (see [below for nested schema](#nestedatt--source--file_source--format_details--parquet))

<a id="nestedatt--source--file_source--format_details--csv"></a>
### Nested Schema for `source.file_source.format_details.csv`

Optional:

- `allow_jagged_rows` (Boolean) Allow rows with fewer columns than expected, fill missing with NULL
- `comment` (String) Lines starting with this character are skipped
- `delimiter` (String) Field separator (e.g., "," or "\t" or "|")
- `encoding` (String) Character encoding (e.g., "UTF8", "ISO-8859-1")
- `escape` (String) Character used to escape special characters
- `has_header` (Boolean) First row contains column names
- `null_values` (List of String) Strings treated as NULL (e.g., ["NULL", "\\N", ""])
- `quote` (String) Character used to quote field values (e.g., "\"")
- `skip_rows` (Number) Number of header rows to skip before data
- `terminator` (String) Line ending (e.g., "\n" or "\r\n")


<a id="nestedatt--source--file_source--format_details--json"></a>
### Nested Schema for `source.file_source.format_details.json`

Optional:

- `flatten` (Boolean)
- `multi_line` (Boolean) Each line contains a separate JSON object (NDJSON/JSONLines format)
- `strip_outer_array` (Boolean) Remove outer array wrapper when parsing


<a id="nestedatt--source--file_source--format_details--parquet"></a>
### Nested Schema for `source.file_source.format_details.parquet`

Optional:

- `binary_as_text` (Boolean) Interpret binary columns as text



<a id="nestedatt--source--file_source--post_processing"></a>
### Nested Schema for `source.file_source.post_processing`

Optional:

- `delete` (Attributes) Delete source file after processing (see [below for nested schema](#nestedatt--source--file_source--post_processing--delete))
- `move` (Attributes) Move source file to another path after processing (see [below for nested schema](#nestedatt--source--file_source--post_processing--move))

<a id="nestedatt--source--file_source--post_processing--delete"></a>
### Nested Schema for `source.file_source.post_processing.delete`


<a id="nestedatt--source--file_source--post_processing--move"></a>
### Nested Schema for `source.file_source.post_processing.move`

Required:

- `destination` (String) Destination path ("s3://bucket/processed/")



<a id="nestedatt--source--file_source--system_columns"></a>
### Nested Schema for `source.file_source.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--file_source--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--file_source--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--file_source--system_columns--synced_at))

<a id="nestedatt--source--file_source--system_columns--lsn"></a>
### Nested Schema for `source.file_source.system_columns.lsn`


<a id="nestedatt--source--file_source--system_columns--op"></a>
### Nested Schema for `source.file_source.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--file_source--system_columns--synced_at"></a>
### Nested Schema for `source.file_source.system_columns.synced_at`



<a id="nestedatt--source--file_source--table_mapping"></a>
### Nested Schema for `source.file_source.table_mapping`

Optional:

- `auto` (Attributes) Each file becomes its own table (see [below for nested schema](#nestedatt--source--file_source--table_mapping--auto))
- `dynamic` (Attributes) Extract table name from path using regex (see [below for nested schema](#nestedatt--source--file_source--table_mapping--dynamic))
- `single` (Attributes) All matched files go to one destination table (see [below for nested schema](#nestedatt--source--file_source--table_mapping--single))

<a id="nestedatt--source--file_source--table_mapping--auto"></a>
### Nested Schema for `source.file_source.table_mapping.auto`

Optional:

- `prefix` (String) Prepend to derived table name
- `suffix` (String) Append to derived table name


<a id="nestedatt--source--file_source--table_mapping--dynamic"></a>
### Nested Schema for `source.file_source.table_mapping.dynamic`

Required:

- `pattern` (String) Regex with named capture groups ("exports/(?P<entity>[^/]+)/(?P<year>[0-9]{4})/.*")
- `template` (String) Template using capture group names ("{entity}_{year}")


<a id="nestedatt--source--file_source--table_mapping--single"></a>
### Nested Schema for `source.file_source.table_mapping.single`

Required:

- `destination` (String) Destination table name ("raw_events")




<a id="nestedatt--source--mongo"></a>
### Nested Schema for `source.mongo`

Required:

- `address` (String)
- `database` (String)
- `replication_type` (Attributes) MongoDB replication method (see [below for nested schema](#nestedatt--source--mongo--replication_type))

Optional:

- `authentication_source` (String)
- `catalog` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--catalog))
- `flatten_body` (Boolean)
- `flatten_max_depth` (Number)
- `max_pool_size` (Number)
- `password` (String, Sensitive)
- `ssl_mode` (Attributes) SSL connection mode for MongoDB (see [below for nested schema](#nestedatt--source--mongo--ssl_mode))
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--mongo--system_columns))
- `user` (String)

<a id="nestedatt--source--mongo--replication_type"></a>
### Nested Schema for `source.mongo.replication_type`

Optional:

- `change_streams` (Attributes) Use Change Streams for CDC (see [below for nested schema](#nestedatt--source--mongo--replication_type--change_streams))
- `snapshot` (Attributes) Initial snapshot only (see [below for nested schema](#nestedatt--source--mongo--replication_type--snapshot))

<a id="nestedatt--source--mongo--replication_type--change_streams"></a>
### Nested Schema for `source.mongo.replication_type.change_streams`

Required:

- `replication_mode` (Attributes) How documents are mapped to target schema (see [below for nested schema](#nestedatt--source--mongo--replication_type--change_streams--replication_mode))

Optional:

- `skip_snapshot` (Boolean) Skip the initial snapshot and start the change stream from the current cluster time

<a id="nestedatt--source--mongo--replication_type--change_streams--replication_mode"></a>
### Nested Schema for `source.mongo.replication_type.change_streams.replication_mode`

Optional:

- `schema_mode` (Attributes) Infer schema from documents, creating typed columns per field (see [below for nested schema](#nestedatt--source--mongo--replication_type--change_streams--replication_mode--schema_mode))
- `schemaless_mode` (Attributes) Store entire documents as JSON in a fixed schema (see [below for nested schema](#nestedatt--source--mongo--replication_type--change_streams--replication_mode--schemaless_mode))

<a id="nestedatt--source--mongo--replication_type--change_streams--replication_mode--schema_mode"></a>
### Nested Schema for `source.mongo.replication_type.change_streams.replication_mode.schema_mode`

Optional:

- `infer_typed_strings` (Boolean) Infer numeric and temporal types from BSON string values (e.g. "123" -> Int64, "2024-01-01" -> Date32)
- `object_store` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--replication_type--change_streams--replication_mode--schema_mode--object_store))

<a id="nestedatt--source--mongo--replication_type--change_streams--replication_mode--schema_mode--object_store"></a>
### Nested Schema for `source.mongo.replication_type.change_streams.replication_mode.schema_mode.object_store`

Required:

- `url` (String) URL: "s3://mybucket", "azure://mycontainer", "gs://mybucket", "file:///absolute/path"

Optional:

- `allow_http` (Boolean) Allow HTTP connections (default: false, HTTPS only)
- `allow_invalid_certificates` (Boolean) Allow invalid/self-signed certificates (default: false)
- `max_concurrent_parts` (Number) Max concurrent part uploads per file. Set to 1 for cross-region or to prevent part upload failures and timeouts due to limited bandwidth.
- `max_concurrent_requests` (Number) Max concurrent requests to the object store. 0 means no limit.
- `options` (Attributes Map) Configuration options (key-value pairs)

S3: [{"name": "region", "value": "us-east-1"}, {"name": "access_key_id", "value": "AKIA..."}, {"name": "secret_access_key", "value": "..."}]

Azure: [{"name": "account_name", "value": "myaccount"}, {"name": "access_key", "value": "..."} or {"name": "sas_token", "value": "sp=..."}]

GCS: [{"name": "service_account_key", "value": "{...JSON...}"}] (see [below for nested schema](#nestedatt--source--mongo--replication_type--change_streams--replication_mode--schema_mode--object_store--options))
- `root_certificate_pem` (String, Sensitive) PEM-encoded root certificate(s) for TLS verification

<a id="nestedatt--source--mongo--replication_type--change_streams--replication_mode--schema_mode--object_store--options"></a>
### Nested Schema for `source.mongo.replication_type.change_streams.replication_mode.schema_mode.object_store.options`

Required:

- `value` (String) Option value




<a id="nestedatt--source--mongo--replication_type--change_streams--replication_mode--schemaless_mode"></a>
### Nested Schema for `source.mongo.replication_type.change_streams.replication_mode.schemaless_mode`




<a id="nestedatt--source--mongo--replication_type--snapshot"></a>
### Nested Schema for `source.mongo.replication_type.snapshot`

Required:

- `replication_mode` (Attributes) How documents are mapped to target schema (see [below for nested schema](#nestedatt--source--mongo--replication_type--snapshot--replication_mode))

<a id="nestedatt--source--mongo--replication_type--snapshot--replication_mode"></a>
### Nested Schema for `source.mongo.replication_type.snapshot.replication_mode`

Optional:

- `schema_mode` (Attributes) Infer schema from documents, creating typed columns per field (see [below for nested schema](#nestedatt--source--mongo--replication_type--snapshot--replication_mode--schema_mode))
- `schemaless_mode` (Attributes) Store entire documents as JSON in a fixed schema (see [below for nested schema](#nestedatt--source--mongo--replication_type--snapshot--replication_mode--schemaless_mode))

<a id="nestedatt--source--mongo--replication_type--snapshot--replication_mode--schema_mode"></a>
### Nested Schema for `source.mongo.replication_type.snapshot.replication_mode.schema_mode`

Optional:

- `infer_typed_strings` (Boolean) Infer numeric and temporal types from BSON string values (e.g. "123" -> Int64, "2024-01-01" -> Date32)
- `object_store` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--replication_type--snapshot--replication_mode--schema_mode--object_store))

<a id="nestedatt--source--mongo--replication_type--snapshot--replication_mode--schema_mode--object_store"></a>
### Nested Schema for `source.mongo.replication_type.snapshot.replication_mode.schema_mode.object_store`

Required:

- `url` (String) URL: "s3://mybucket", "azure://mycontainer", "gs://mybucket", "file:///absolute/path"

Optional:

- `allow_http` (Boolean) Allow HTTP connections (default: false, HTTPS only)
- `allow_invalid_certificates` (Boolean) Allow invalid/self-signed certificates (default: false)
- `max_concurrent_parts` (Number) Max concurrent part uploads per file. Set to 1 for cross-region or to prevent part upload failures and timeouts due to limited bandwidth.
- `max_concurrent_requests` (Number) Max concurrent requests to the object store. 0 means no limit.
- `options` (Attributes Map) Configuration options (key-value pairs)

S3: [{"name": "region", "value": "us-east-1"}, {"name": "access_key_id", "value": "AKIA..."}, {"name": "secret_access_key", "value": "..."}]

Azure: [{"name": "account_name", "value": "myaccount"}, {"name": "access_key", "value": "..."} or {"name": "sas_token", "value": "sp=..."}]

GCS: [{"name": "service_account_key", "value": "{...JSON...}"}] (see [below for nested schema](#nestedatt--source--mongo--replication_type--snapshot--replication_mode--schema_mode--object_store--options))
- `root_certificate_pem` (String, Sensitive) PEM-encoded root certificate(s) for TLS verification

<a id="nestedatt--source--mongo--replication_type--snapshot--replication_mode--schema_mode--object_store--options"></a>
### Nested Schema for `source.mongo.replication_type.snapshot.replication_mode.schema_mode.object_store.options`

Required:

- `value` (String) Option value




<a id="nestedatt--source--mongo--replication_type--snapshot--replication_mode--schemaless_mode"></a>
### Nested Schema for `source.mongo.replication_type.snapshot.replication_mode.schemaless_mode`





<a id="nestedatt--source--mongo--catalog"></a>
### Nested Schema for `source.mongo.catalog`

Required:

- `name` (String)
- `schemas` (Attributes Map) (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas))

Optional:

- `default_action` (String)

<a id="nestedatt--source--mongo--catalog--schemas"></a>
### Nested Schema for `source.mongo.catalog.schemas`

Required:

- `tables` (Attributes Map) (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables))

Optional:

- `action` (String)

<a id="nestedatt--source--mongo--catalog--schemas--tables"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables`

Optional:

- `action` (String)
- `columns` (Attributes Map) (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--columns))
- `iceberg_partition_spec` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec))

<a id="nestedatt--source--mongo--catalog--schemas--tables--columns"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.columns`

Optional:

- `action` (String)


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec`

Required:

- `fields` (Attributes List) Partition fields, applied in order (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields))

<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields`

Required:

- `source_column` (String) Source column name
- `transform` (Attributes) Transform applied to the source column (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform))

Optional:

- `name` (String) Partition column name in Iceberg. Defaults to {source_column}_{transform} (e.g. created_at_day)

<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform`

Optional:

- `bucket` (Attributes) Hash into a fixed number of buckets (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket))
- `day` (Attributes) Day of a date or timestamp (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day))
- `hour` (Attributes) Hour of a timestamp (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour))
- `identity` (Attributes) Source value, unchanged (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity))
- `month` (Attributes) Month of a date or timestamp (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month))
- `truncate` (Attributes) Truncate to a fixed width (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate))
- `year` (Attributes) Year of a date or timestamp (see [below for nested schema](#nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year))

<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.bucket`

Optional:

- `num_buckets` (Number) Number of hash buckets


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.day`


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.hour`


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.identity`


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.month`


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.truncate`

Optional:

- `width` (Number) Truncation width: characters for strings, bytes for binary, modulus for integers and decimals


<a id="nestedatt--source--mongo--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year"></a>
### Nested Schema for `source.mongo.catalog.schemas.tables.iceberg_partition_spec.fields.transform.year`








<a id="nestedatt--source--mongo--ssl_mode"></a>
### Nested Schema for `source.mongo.ssl_mode`

Optional:

- `disable` (Attributes) Disable SSL and use an unencrypted connection (see [below for nested schema](#nestedatt--source--mongo--ssl_mode--disable))
- `enable` (Attributes) Enable SSL encryption for the connection (see [below for nested schema](#nestedatt--source--mongo--ssl_mode--enable))

<a id="nestedatt--source--mongo--ssl_mode--disable"></a>
### Nested Schema for `source.mongo.ssl_mode.disable`


<a id="nestedatt--source--mongo--ssl_mode--enable"></a>
### Nested Schema for `source.mongo.ssl_mode.enable`

Optional:

- `ca_file` (String, Sensitive) PEM-encoded CA certificate content for server verification
- `cert_key_file` (String, Sensitive) PEM-encoded client certificate and private key content for mutual TLS



<a id="nestedatt--source--mongo--system_columns"></a>
### Nested Schema for `source.mongo.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--mongo--system_columns--synced_at))

<a id="nestedatt--source--mongo--system_columns--lsn"></a>
### Nested Schema for `source.mongo.system_columns.lsn`


<a id="nestedatt--source--mongo--system_columns--op"></a>
### Nested Schema for `source.mongo.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--mongo--system_columns--synced_at"></a>
### Nested Schema for `source.mongo.system_columns.synced_at`




<a id="nestedatt--source--mysql"></a>
### Nested Schema for `source.mysql`

Required:

- `host` (String)

Optional:

- `catalog` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--catalog))
- `database` (String)
- `infer_tinyint1_as_boolean` (Boolean)
- `keyless_table_strategy` (Attributes) How to replicate tables without a primary key (see [below for nested schema](#nestedatt--source--mysql--keyless_table_strategy))
- `max_pool_size` (Number)
- `parallel_snapshots_enabled` (Boolean)
- `password` (String, Sensitive)
- `port` (Number)
- `skip_snapshot` (Boolean)
- `ssl_mode` (Attributes) SSL encryption mode for the connection to MySQL (see [below for nested schema](#nestedatt--source--mysql--ssl_mode))
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--mysql--system_columns))
- `tunnel` (Attributes) Optional network transport. Leave unset for direct TCP; pick a
 variant to tunnel the connection. Today only SSH bastion is supported;
 additional transports (e.g. PrivateLink) can be added as new variants. (see [below for nested schema](#nestedatt--source--mysql--tunnel))
- `user` (String)

<a id="nestedatt--source--mysql--catalog"></a>
### Nested Schema for `source.mysql.catalog`

Required:

- `name` (String)
- `schemas` (Attributes Map) (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas))

Optional:

- `default_action` (String)

<a id="nestedatt--source--mysql--catalog--schemas"></a>
### Nested Schema for `source.mysql.catalog.schemas`

Required:

- `tables` (Attributes Map) (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables))

Optional:

- `action` (String)

<a id="nestedatt--source--mysql--catalog--schemas--tables"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables`

Optional:

- `action` (String)
- `columns` (Attributes Map) (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--columns))
- `iceberg_partition_spec` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec))

<a id="nestedatt--source--mysql--catalog--schemas--tables--columns"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.columns`

Optional:

- `action` (String)


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec`

Required:

- `fields` (Attributes List) Partition fields, applied in order (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields))

<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields`

Required:

- `source_column` (String) Source column name
- `transform` (Attributes) Transform applied to the source column (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform))

Optional:

- `name` (String) Partition column name in Iceberg. Defaults to {source_column}_{transform} (e.g. created_at_day)

<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform`

Optional:

- `bucket` (Attributes) Hash into a fixed number of buckets (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket))
- `day` (Attributes) Day of a date or timestamp (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day))
- `hour` (Attributes) Hour of a timestamp (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour))
- `identity` (Attributes) Source value, unchanged (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity))
- `month` (Attributes) Month of a date or timestamp (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month))
- `truncate` (Attributes) Truncate to a fixed width (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate))
- `year` (Attributes) Year of a date or timestamp (see [below for nested schema](#nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year))

<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.bucket`

Optional:

- `num_buckets` (Number) Number of hash buckets


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.day`


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.hour`


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.identity`


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.month`


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.truncate`

Optional:

- `width` (Number) Truncation width: characters for strings, bytes for binary, modulus for integers and decimals


<a id="nestedatt--source--mysql--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year"></a>
### Nested Schema for `source.mysql.catalog.schemas.tables.iceberg_partition_spec.fields.transform.year`








<a id="nestedatt--source--mysql--keyless_table_strategy"></a>
### Nested Schema for `source.mysql.keyless_table_strategy`

Optional:

- `append_only` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--keyless_table_strategy--append_only))
- `dedup_by_row_hash` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--keyless_table_strategy--dedup_by_row_hash))

<a id="nestedatt--source--mysql--keyless_table_strategy--append_only"></a>
### Nested Schema for `source.mysql.keyless_table_strategy.append_only`


<a id="nestedatt--source--mysql--keyless_table_strategy--dedup_by_row_hash"></a>
### Nested Schema for `source.mysql.keyless_table_strategy.dedup_by_row_hash`



<a id="nestedatt--source--mysql--ssl_mode"></a>
### Nested Schema for `source.mysql.ssl_mode`

Optional:

- `disable` (Attributes) Disable SSL encryption for the connection (see [below for nested schema](#nestedatt--source--mysql--ssl_mode--disable))
- `require` (Attributes) Require SSL encryption but do not verify the server certificate (see [below for nested schema](#nestedatt--source--mysql--ssl_mode--require))
- `verify_ca` (Attributes) Require SSL encryption and verify the server certificate against the CA (see [below for nested schema](#nestedatt--source--mysql--ssl_mode--verify_ca))
- `verify_identity` (Attributes) Require SSL encryption, verify the server certificate, and verify the server hostname (see [below for nested schema](#nestedatt--source--mysql--ssl_mode--verify_identity))

<a id="nestedatt--source--mysql--ssl_mode--disable"></a>
### Nested Schema for `source.mysql.ssl_mode.disable`


<a id="nestedatt--source--mysql--ssl_mode--require"></a>
### Nested Schema for `source.mysql.ssl_mode.require`


<a id="nestedatt--source--mysql--ssl_mode--verify_ca"></a>
### Nested Schema for `source.mysql.ssl_mode.verify_ca`

Optional:

- `ca_file` (String, Sensitive) CA certificate content for verifying the server's SSL certificate


<a id="nestedatt--source--mysql--ssl_mode--verify_identity"></a>
### Nested Schema for `source.mysql.ssl_mode.verify_identity`

Optional:

- `ca_file` (String, Sensitive) CA certificate content for verifying the server's SSL certificate and hostname



<a id="nestedatt--source--mysql--system_columns"></a>
### Nested Schema for `source.mysql.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--mysql--system_columns--synced_at))

<a id="nestedatt--source--mysql--system_columns--lsn"></a>
### Nested Schema for `source.mysql.system_columns.lsn`


<a id="nestedatt--source--mysql--system_columns--op"></a>
### Nested Schema for `source.mysql.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--mysql--system_columns--synced_at"></a>
### Nested Schema for `source.mysql.system_columns.synced_at`



<a id="nestedatt--source--mysql--tunnel"></a>
### Nested Schema for `source.mysql.tunnel`

Optional:

- `ssh` (Attributes) Tunnel through an SSH bastion host (see [below for nested schema](#nestedatt--source--mysql--tunnel--ssh))

<a id="nestedatt--source--mysql--tunnel--ssh"></a>
### Nested Schema for `source.mysql.tunnel.ssh`

Required:

- `auth` (Attributes) How to authenticate against the bastion (see [below for nested schema](#nestedatt--source--mysql--tunnel--ssh--auth))
- `bastion_host` (String) Hostname or IP of the SSH bastion server
- `user` (String) SSH username on the bastion server

Optional:

- `bastion_alternates` (List of String) Fallback bastion hostnames, tried in order if the primary is unreachable
- `bastion_port` (Number) SSH port on the bastion server

<a id="nestedatt--source--mysql--tunnel--ssh--auth"></a>
### Nested Schema for `source.mysql.tunnel.ssh.auth`

Optional:

- `bring_your_own_key` (Attributes) Paste your own private key (see [below for nested schema](#nestedatt--source--mysql--tunnel--ssh--auth--bring_your_own_key))
- `generated_key` (Attributes) Supermetal generates the keypair; you install the public key on the bastion (see [below for nested schema](#nestedatt--source--mysql--tunnel--ssh--auth--generated_key))

<a id="nestedatt--source--mysql--tunnel--ssh--auth--bring_your_own_key"></a>
### Nested Schema for `source.mysql.tunnel.ssh.auth.bring_your_own_key`

Required:

- `private_key` (String, Sensitive) OpenSSH-encoded private key


<a id="nestedatt--source--mysql--tunnel--ssh--auth--generated_key"></a>
### Nested Schema for `source.mysql.tunnel.ssh.auth.generated_key`

Required:

- `private_key` (String, Sensitive) Private key (managed by Supermetal)
- `public_key` (String) Public key — add this line to ~/.ssh/authorized_keys on your bastion






<a id="nestedatt--source--oracle"></a>
### Nested Schema for `source.oracle`

Required:

- `connect_string` (String)
- `password` (String, Sensitive)
- `replication_type` (Attributes) Oracle replication method (see [below for nested schema](#nestedatt--source--oracle--replication_type))
- `user` (String)

Optional:

- `catalog` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--catalog))
- `keyless_table_strategy` (Attributes) How to replicate tables without a primary key (see [below for nested schema](#nestedatt--source--oracle--keyless_table_strategy))
- `max_pool_size` (Number)
- `parallel_snapshots_enabled` (Boolean)
- `pdb` (String)
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--oracle--system_columns))

<a id="nestedatt--source--oracle--replication_type"></a>
### Nested Schema for `source.oracle.replication_type`

Optional:

- `log_miner` (Attributes) Capture data changes using Oracle LogMiner (see [below for nested schema](#nestedatt--source--oracle--replication_type--log_miner))

<a id="nestedatt--source--oracle--replication_type--log_miner"></a>
### Nested Schema for `source.oracle.replication_type.log_miner`

Optional:

- `committed_data_only` (Boolean) Only capture committed transactions from LogMiner sessions
- `dict` (String)
- `max_buffer_memory_size_per_tx` (Number) Max memory per transaction before flushing (bytes)
- `poll_interval_secs` (Number) How often to poll for new changes (seconds)



<a id="nestedatt--source--oracle--catalog"></a>
### Nested Schema for `source.oracle.catalog`

Required:

- `name` (String)
- `schemas` (Attributes Map) (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas))

Optional:

- `default_action` (String)

<a id="nestedatt--source--oracle--catalog--schemas"></a>
### Nested Schema for `source.oracle.catalog.schemas`

Required:

- `tables` (Attributes Map) (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables))

Optional:

- `action` (String)

<a id="nestedatt--source--oracle--catalog--schemas--tables"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables`

Optional:

- `action` (String)
- `columns` (Attributes Map) (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--columns))
- `iceberg_partition_spec` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec))

<a id="nestedatt--source--oracle--catalog--schemas--tables--columns"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.columns`

Optional:

- `action` (String)


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec`

Required:

- `fields` (Attributes List) Partition fields, applied in order (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields))

<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields`

Required:

- `source_column` (String) Source column name
- `transform` (Attributes) Transform applied to the source column (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform))

Optional:

- `name` (String) Partition column name in Iceberg. Defaults to {source_column}_{transform} (e.g. created_at_day)

<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform`

Optional:

- `bucket` (Attributes) Hash into a fixed number of buckets (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket))
- `day` (Attributes) Day of a date or timestamp (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day))
- `hour` (Attributes) Hour of a timestamp (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour))
- `identity` (Attributes) Source value, unchanged (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity))
- `month` (Attributes) Month of a date or timestamp (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month))
- `truncate` (Attributes) Truncate to a fixed width (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate))
- `year` (Attributes) Year of a date or timestamp (see [below for nested schema](#nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year))

<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.bucket`

Optional:

- `num_buckets` (Number) Number of hash buckets


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.day`


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.hour`


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.identity`


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.month`


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.truncate`

Optional:

- `width` (Number) Truncation width: characters for strings, bytes for binary, modulus for integers and decimals


<a id="nestedatt--source--oracle--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year"></a>
### Nested Schema for `source.oracle.catalog.schemas.tables.iceberg_partition_spec.fields.transform.year`








<a id="nestedatt--source--oracle--keyless_table_strategy"></a>
### Nested Schema for `source.oracle.keyless_table_strategy`

Optional:

- `append_only` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--keyless_table_strategy--append_only))
- `dedup_by_row_hash` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--keyless_table_strategy--dedup_by_row_hash))

<a id="nestedatt--source--oracle--keyless_table_strategy--append_only"></a>
### Nested Schema for `source.oracle.keyless_table_strategy.append_only`


<a id="nestedatt--source--oracle--keyless_table_strategy--dedup_by_row_hash"></a>
### Nested Schema for `source.oracle.keyless_table_strategy.dedup_by_row_hash`



<a id="nestedatt--source--oracle--system_columns"></a>
### Nested Schema for `source.oracle.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--oracle--system_columns--synced_at))

<a id="nestedatt--source--oracle--system_columns--lsn"></a>
### Nested Schema for `source.oracle.system_columns.lsn`


<a id="nestedatt--source--oracle--system_columns--op"></a>
### Nested Schema for `source.oracle.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--oracle--system_columns--synced_at"></a>
### Nested Schema for `source.oracle.system_columns.synced_at`




<a id="nestedatt--source--postgres"></a>
### Nested Schema for `source.postgres`

Required:

- `database` (String)
- `host` (String)
- `password` (String, Sensitive)
- `replication_type` (Attributes) Specifies the PostgreSQL replication method (see [below for nested schema](#nestedatt--source--postgres--replication_type))
- `user` (String)

Optional:

- `catalog` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--catalog))
- `keyless_table_strategy` (Attributes) How to replicate tables without a primary key (see [below for nested schema](#nestedatt--source--postgres--keyless_table_strategy))
- `max_pool_size` (Number)
- `operation_lock_timeout_seconds` (Number)
- `parallel_snapshots_enabled` (Boolean)
- `partitions_as_root` (Boolean)
- `port` (Number)
- `ssl_cert` (String, Sensitive)
- `ssl_key` (String, Sensitive)
- `ssl_mode` (String) SSL connection mode for the PostgreSQL server
- `ssl_root_cert` (String, Sensitive)
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--postgres--system_columns))
- `tunnel` (Attributes) Optional network transport. Leave unset for direct TCP; pick a
 variant to tunnel the connection. Today only SSH bastion is supported;
 additional transports (e.g. PrivateLink) can be added as new variants. (see [below for nested schema](#nestedatt--source--postgres--tunnel))

<a id="nestedatt--source--postgres--replication_type"></a>
### Nested Schema for `source.postgres.replication_type`

Optional:

- `logical_replication` (Attributes) Use logical replication (see [below for nested schema](#nestedatt--source--postgres--replication_type--logical_replication))
- `query_based` (Attributes) Use xmin column for change detection without publications or replication slots (see [below for nested schema](#nestedatt--source--postgres--replication_type--query_based))
- `snapshot` (Attributes) Initial snapshot / backfill only (see [below for nested schema](#nestedatt--source--postgres--replication_type--snapshot))

<a id="nestedatt--source--postgres--replication_type--logical_replication"></a>
### Nested Schema for `source.postgres.replication_type.logical_replication`

Optional:

- `publication_name` (String) Existing publication to subscribe to. Superusers can leave this empty to create one automatically.
- `retry_window_seconds` (Number) Maximum time window (in seconds) to retry transient connection errors before failing. Defaults to 300 seconds.
- `skip_snapshots` (Boolean) Skip the initial snapshot/backfill and start streaming changes from the replication slot's consistent point. Use when existing data does not need to be replicated.


<a id="nestedatt--source--postgres--replication_type--query_based"></a>
### Nested Schema for `source.postgres.replication_type.query_based`

Optional:

- `delete_tracking` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--replication_type--query_based--delete_tracking))
- `max_skip_cycles` (Number) Sync cycles a table may skip while statistics show no change. After this many skips it scans anyway to catch lagging counters. Set to 0 to scan every table on every cycle.
- `poll_interval_seconds` (Number) Seconds between sync cycles for inserts and updates

<a id="nestedatt--source--postgres--replication_type--query_based--delete_tracking"></a>
### Nested Schema for `source.postgres.replication_type.query_based.delete_tracking`

Optional:

- `enabled` (Attributes) Track deletes by storing primary keys locally (see [below for nested schema](#nestedatt--source--postgres--replication_type--query_based--delete_tracking--enabled))

<a id="nestedatt--source--postgres--replication_type--query_based--delete_tracking--enabled"></a>
### Nested Schema for `source.postgres.replication_type.query_based.delete_tracking.enabled`

Optional:

- `pk_index_store` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--replication_type--query_based--delete_tracking--enabled--pk_index_store))
- `poll_interval_seconds` (Number) Seconds between delete detection cycles. Set to 0 to run every sync cycle.

<a id="nestedatt--source--postgres--replication_type--query_based--delete_tracking--enabled--pk_index_store"></a>
### Nested Schema for `source.postgres.replication_type.query_based.delete_tracking.enabled.pk_index_store`

Required:

- `url` (String) URL: "s3://mybucket", "azure://mycontainer", "gs://mybucket", "file:///absolute/path"

Optional:

- `allow_http` (Boolean) Allow HTTP connections (default: false, HTTPS only)
- `allow_invalid_certificates` (Boolean) Allow invalid/self-signed certificates (default: false)
- `max_concurrent_parts` (Number) Max concurrent part uploads per file. Set to 1 for cross-region or to prevent part upload failures and timeouts due to limited bandwidth.
- `max_concurrent_requests` (Number) Max concurrent requests to the object store. 0 means no limit.
- `options` (Attributes Map) Configuration options (key-value pairs)

S3: [{"name": "region", "value": "us-east-1"}, {"name": "access_key_id", "value": "AKIA..."}, {"name": "secret_access_key", "value": "..."}]

Azure: [{"name": "account_name", "value": "myaccount"}, {"name": "access_key", "value": "..."} or {"name": "sas_token", "value": "sp=..."}]

GCS: [{"name": "service_account_key", "value": "{...JSON...}"}] (see [below for nested schema](#nestedatt--source--postgres--replication_type--query_based--delete_tracking--enabled--pk_index_store--options))
- `root_certificate_pem` (String, Sensitive) PEM-encoded root certificate(s) for TLS verification

<a id="nestedatt--source--postgres--replication_type--query_based--delete_tracking--enabled--pk_index_store--options"></a>
### Nested Schema for `source.postgres.replication_type.query_based.delete_tracking.enabled.pk_index_store.options`

Required:

- `value` (String) Option value






<a id="nestedatt--source--postgres--replication_type--snapshot"></a>
### Nested Schema for `source.postgres.replication_type.snapshot`



<a id="nestedatt--source--postgres--catalog"></a>
### Nested Schema for `source.postgres.catalog`

Required:

- `name` (String)
- `schemas` (Attributes Map) (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas))

Optional:

- `default_action` (String)

<a id="nestedatt--source--postgres--catalog--schemas"></a>
### Nested Schema for `source.postgres.catalog.schemas`

Required:

- `tables` (Attributes Map) (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables))

Optional:

- `action` (String)

<a id="nestedatt--source--postgres--catalog--schemas--tables"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables`

Optional:

- `action` (String)
- `columns` (Attributes Map) (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--columns))
- `iceberg_partition_spec` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec))

<a id="nestedatt--source--postgres--catalog--schemas--tables--columns"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.columns`

Optional:

- `action` (String)


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec`

Required:

- `fields` (Attributes List) Partition fields, applied in order (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields))

<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields`

Required:

- `source_column` (String) Source column name
- `transform` (Attributes) Transform applied to the source column (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform))

Optional:

- `name` (String) Partition column name in Iceberg. Defaults to {source_column}_{transform} (e.g. created_at_day)

<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform`

Optional:

- `bucket` (Attributes) Hash into a fixed number of buckets (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket))
- `day` (Attributes) Day of a date or timestamp (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day))
- `hour` (Attributes) Hour of a timestamp (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour))
- `identity` (Attributes) Source value, unchanged (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity))
- `month` (Attributes) Month of a date or timestamp (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month))
- `truncate` (Attributes) Truncate to a fixed width (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate))
- `year` (Attributes) Year of a date or timestamp (see [below for nested schema](#nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year))

<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.bucket`

Optional:

- `num_buckets` (Number) Number of hash buckets


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.day`


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.hour`


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.identity`


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.month`


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.truncate`

Optional:

- `width` (Number) Truncation width: characters for strings, bytes for binary, modulus for integers and decimals


<a id="nestedatt--source--postgres--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year"></a>
### Nested Schema for `source.postgres.catalog.schemas.tables.iceberg_partition_spec.fields.transform.year`








<a id="nestedatt--source--postgres--keyless_table_strategy"></a>
### Nested Schema for `source.postgres.keyless_table_strategy`

Optional:

- `append_only` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--keyless_table_strategy--append_only))
- `dedup_by_row_hash` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--keyless_table_strategy--dedup_by_row_hash))

<a id="nestedatt--source--postgres--keyless_table_strategy--append_only"></a>
### Nested Schema for `source.postgres.keyless_table_strategy.append_only`


<a id="nestedatt--source--postgres--keyless_table_strategy--dedup_by_row_hash"></a>
### Nested Schema for `source.postgres.keyless_table_strategy.dedup_by_row_hash`



<a id="nestedatt--source--postgres--system_columns"></a>
### Nested Schema for `source.postgres.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--postgres--system_columns--synced_at))

<a id="nestedatt--source--postgres--system_columns--lsn"></a>
### Nested Schema for `source.postgres.system_columns.lsn`


<a id="nestedatt--source--postgres--system_columns--op"></a>
### Nested Schema for `source.postgres.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--postgres--system_columns--synced_at"></a>
### Nested Schema for `source.postgres.system_columns.synced_at`



<a id="nestedatt--source--postgres--tunnel"></a>
### Nested Schema for `source.postgres.tunnel`

Optional:

- `ssh` (Attributes) Tunnel through an SSH bastion host (see [below for nested schema](#nestedatt--source--postgres--tunnel--ssh))

<a id="nestedatt--source--postgres--tunnel--ssh"></a>
### Nested Schema for `source.postgres.tunnel.ssh`

Required:

- `auth` (Attributes) How to authenticate against the bastion (see [below for nested schema](#nestedatt--source--postgres--tunnel--ssh--auth))
- `bastion_host` (String) Hostname or IP of the SSH bastion server
- `user` (String) SSH username on the bastion server

Optional:

- `bastion_alternates` (List of String) Fallback bastion hostnames, tried in order if the primary is unreachable
- `bastion_port` (Number) SSH port on the bastion server

<a id="nestedatt--source--postgres--tunnel--ssh--auth"></a>
### Nested Schema for `source.postgres.tunnel.ssh.auth`

Optional:

- `bring_your_own_key` (Attributes) Paste your own private key (see [below for nested schema](#nestedatt--source--postgres--tunnel--ssh--auth--bring_your_own_key))
- `generated_key` (Attributes) Supermetal generates the keypair; you install the public key on the bastion (see [below for nested schema](#nestedatt--source--postgres--tunnel--ssh--auth--generated_key))

<a id="nestedatt--source--postgres--tunnel--ssh--auth--bring_your_own_key"></a>
### Nested Schema for `source.postgres.tunnel.ssh.auth.bring_your_own_key`

Required:

- `private_key` (String, Sensitive) OpenSSH-encoded private key


<a id="nestedatt--source--postgres--tunnel--ssh--auth--generated_key"></a>
### Nested Schema for `source.postgres.tunnel.ssh.auth.generated_key`

Required:

- `private_key` (String, Sensitive) Private key (managed by Supermetal)
- `public_key` (String) Public key — add this line to ~/.ssh/authorized_keys on your bastion






<a id="nestedatt--source--sqlserver"></a>
### Nested Schema for `source.sqlserver`

Required:

- `auth` (Attributes) Authentication method for SQL Server (see [below for nested schema](#nestedatt--source--sqlserver--auth))
- `database` (String)
- `host` (String)
- `replication_type` (Attributes) SQL Server replication method (see [below for nested schema](#nestedatt--source--sqlserver--replication_type))

Optional:

- `catalog` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--catalog))
- `keyless_table_strategy` (Attributes) How to replicate tables without a primary key (see [below for nested schema](#nestedatt--source--sqlserver--keyless_table_strategy))
- `max_pool_size` (Number)
- `parallel_snapshots_enabled` (Boolean)
- `port` (Number)
- `snapshot_isolation` (String) Snapshot isolation behavior during snapshot reads
- `ssl_mode` (Attributes) SSL encryption mode for the connection to SQL Server (see [below for nested schema](#nestedatt--source--sqlserver--ssl_mode))
- `system_columns` (Attributes) Optional metadata columns to append to every row (e.g. `_sm_synced_at`) (see [below for nested schema](#nestedatt--source--sqlserver--system_columns))

<a id="nestedatt--source--sqlserver--auth"></a>
### Nested Schema for `source.sqlserver.auth`

Optional:

- `activedirectory` (Attributes) Azure Active Directory Authentication (access token) (see [below for nested schema](#nestedatt--source--sqlserver--auth--activedirectory))
- `standard` (Attributes) SQL Server Authentication (username and password) (see [below for nested schema](#nestedatt--source--sqlserver--auth--standard))
- `windows` (Attributes) Windows Authentication (username and password) (see [below for nested schema](#nestedatt--source--sqlserver--auth--windows))

<a id="nestedatt--source--sqlserver--auth--activedirectory"></a>
### Nested Schema for `source.sqlserver.auth.activedirectory`

Required:

- `token` (String, Sensitive) Azure Active Directory access token


<a id="nestedatt--source--sqlserver--auth--standard"></a>
### Nested Schema for `source.sqlserver.auth.standard`

Required:

- `password` (String, Sensitive) SQL Server login password
- `user` (String) SQL Server login username


<a id="nestedatt--source--sqlserver--auth--windows"></a>
### Nested Schema for `source.sqlserver.auth.windows`

Required:

- `password` (String, Sensitive) Windows password
- `user` (String) Windows username (DOMAIN\\user or user@fully.qualified.domain)



<a id="nestedatt--source--sqlserver--replication_type"></a>
### Nested Schema for `source.sqlserver.replication_type`

Optional:

- `capture_table_replication` (Attributes) Use Change Data Capture (CDC) for ongoing replication (see [below for nested schema](#nestedatt--source--sqlserver--replication_type--capture_table_replication))
- `snapshot` (Attributes) Initial snapshot / backfill only (see [below for nested schema](#nestedatt--source--sqlserver--replication_type--snapshot))

<a id="nestedatt--source--sqlserver--replication_type--capture_table_replication"></a>
### Nested Schema for `source.sqlserver.replication_type.capture_table_replication`

Optional:

- `cdc_max_scans` (Number) Maximum number of scan cycles per polling interval for CDC
- `cdc_max_trans` (Number) Maximum number of transactions to process in a single polling cycle for CDC
- `cdc_poll_interval_secs` (Number) Polling interval in seconds to check for new CDC changes
- `skip_snapshot` (Boolean) Skip the initial snapshot and start CDC from the current max LSN
- `skip_tables_without_ct` (Boolean) Skip syncing tables without CDC capture instances instead of failing
- `transactional_cdc_enabled` (Boolean) Enable transactionally consistent CDC processing


<a id="nestedatt--source--sqlserver--replication_type--snapshot"></a>
### Nested Schema for `source.sqlserver.replication_type.snapshot`



<a id="nestedatt--source--sqlserver--catalog"></a>
### Nested Schema for `source.sqlserver.catalog`

Required:

- `name` (String)
- `schemas` (Attributes Map) (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas))

Optional:

- `default_action` (String)

<a id="nestedatt--source--sqlserver--catalog--schemas"></a>
### Nested Schema for `source.sqlserver.catalog.schemas`

Required:

- `tables` (Attributes Map) (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables))

Optional:

- `action` (String)

<a id="nestedatt--source--sqlserver--catalog--schemas--tables"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables`

Optional:

- `action` (String)
- `columns` (Attributes Map) (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--columns))
- `iceberg_partition_spec` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec))

<a id="nestedatt--source--sqlserver--catalog--schemas--tables--columns"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.columns`

Optional:

- `action` (String)


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec`

Required:

- `fields` (Attributes List) Partition fields, applied in order (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields))

<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields`

Required:

- `source_column` (String) Source column name
- `transform` (Attributes) Transform applied to the source column (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform))

Optional:

- `name` (String) Partition column name in Iceberg. Defaults to {source_column}_{transform} (e.g. created_at_day)

<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform`

Optional:

- `bucket` (Attributes) Hash into a fixed number of buckets (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket))
- `day` (Attributes) Day of a date or timestamp (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day))
- `hour` (Attributes) Hour of a timestamp (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour))
- `identity` (Attributes) Source value, unchanged (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity))
- `month` (Attributes) Month of a date or timestamp (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month))
- `truncate` (Attributes) Truncate to a fixed width (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate))
- `year` (Attributes) Year of a date or timestamp (see [below for nested schema](#nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year))

<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--bucket"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.bucket`

Optional:

- `num_buckets` (Number) Number of hash buckets


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--day"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.day`


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--hour"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.hour`


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--identity"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.identity`


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--month"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.month`


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--truncate"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.truncate`

Optional:

- `width` (Number) Truncation width: characters for strings, bytes for binary, modulus for integers and decimals


<a id="nestedatt--source--sqlserver--catalog--schemas--tables--iceberg_partition_spec--fields--transform--year"></a>
### Nested Schema for `source.sqlserver.catalog.schemas.tables.iceberg_partition_spec.fields.transform.year`








<a id="nestedatt--source--sqlserver--keyless_table_strategy"></a>
### Nested Schema for `source.sqlserver.keyless_table_strategy`

Optional:

- `append_only` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--keyless_table_strategy--append_only))
- `dedup_by_row_hash` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--keyless_table_strategy--dedup_by_row_hash))

<a id="nestedatt--source--sqlserver--keyless_table_strategy--append_only"></a>
### Nested Schema for `source.sqlserver.keyless_table_strategy.append_only`


<a id="nestedatt--source--sqlserver--keyless_table_strategy--dedup_by_row_hash"></a>
### Nested Schema for `source.sqlserver.keyless_table_strategy.dedup_by_row_hash`



<a id="nestedatt--source--sqlserver--ssl_mode"></a>
### Nested Schema for `source.sqlserver.ssl_mode`

Optional:

- `disable` (Attributes) Disable SSL encryption for the connection (see [below for nested schema](#nestedatt--source--sqlserver--ssl_mode--disable))
- `enable` (Attributes) Enable SSL encryption for the connection (see [below for nested schema](#nestedatt--source--sqlserver--ssl_mode--enable))

<a id="nestedatt--source--sqlserver--ssl_mode--disable"></a>
### Nested Schema for `source.sqlserver.ssl_mode.disable`


<a id="nestedatt--source--sqlserver--ssl_mode--enable"></a>
### Nested Schema for `source.sqlserver.ssl_mode.enable`

Optional:

- `ca_file` (String, Sensitive) CA certificate content for verifying the server's SSL certificate



<a id="nestedatt--source--sqlserver--system_columns"></a>
### Nested Schema for `source.sqlserver.system_columns`

Optional:

- `lsn` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--system_columns--lsn))
- `op` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--system_columns--op))
- `synced_at` (Attributes) (see [below for nested schema](#nestedatt--source--sqlserver--system_columns--synced_at))

<a id="nestedatt--source--sqlserver--system_columns--lsn"></a>
### Nested Schema for `source.sqlserver.system_columns.lsn`


<a id="nestedatt--source--sqlserver--system_columns--op"></a>
### Nested Schema for `source.sqlserver.system_columns.op`

Optional:

- `encoding` (String)
- `snapshot_as_read` (Boolean) Show snapshot rows as read (r/4/read) instead of insert


<a id="nestedatt--source--sqlserver--system_columns--synced_at"></a>
### Nested Schema for `source.sqlserver.system_columns.synced_at`

## Import

```shell
#!/bin/bash
# 1. Write the resource block with full config (including secrets)
# 2. Run the import command below
# 3. Run terraform plan (will show secrets being set)
# 4. Run terraform apply (converges state)

terraform import supermetal_connector.quickstart quickstart
```
