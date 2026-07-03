terraform {
  required_providers {
    supermetal = {
      source = "supermetal-inc/supermetal"
    }
  }
}

variable "supermetal_username" {
  type      = string
  sensitive = true
}

variable "supermetal_password" {
  type      = string
  sensitive = true
}

provider "supermetal" {
  endpoint = "https://sm.internal:3000"
  username = var.supermetal_username
  password = var.supermetal_password
}

# Multiple agents via provider aliases
provider "supermetal" {
  alias    = "analytics"
  endpoint = "https://sm-analytics.internal:3000"
  username = var.supermetal_username
  password = var.supermetal_password
}
