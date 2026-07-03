#!/bin/bash
# 1. Write the resource block with full config (including secrets)
# 2. Run the import command below
# 3. Run terraform plan (will show secrets being set)
# 4. Run terraform apply (converges state)

terraform import supermetal_connector.quickstart quickstart
