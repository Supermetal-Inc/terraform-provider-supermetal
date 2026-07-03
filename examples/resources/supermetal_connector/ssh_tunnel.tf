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
