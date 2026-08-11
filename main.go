package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/provider"
)

// The documentation generator installs providers under the HashiCorp namespace
// while exporting schemas. This override applies only to documentation builds.
// Production binaries continue to advertise providerAddress.
//go:generate sh -c "GOFLAGS='-ldflags=-X=main.providerAddress=registry.terraform.io/hashicorp/supermetal' go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate --provider-name supermetal"

var (
	version         = "dev"
	providerAddress = "registry.terraform.io/supermetal-inc/supermetal"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: providerAddress,
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
