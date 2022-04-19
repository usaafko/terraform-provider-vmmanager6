package main

import (
	"flag"

	"github.com/usaafko/terraform-provider-vmmanager6/vmmanager6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
        "github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {

        var debugMode bool
        var pluginPath string

        flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.StringVar(&pluginPath, "registry", "github.com/usaafko/terraform-provider-vmmanager6", "specify path, useful for local debugging")
        flag.Parse()

        opts := &plugin.ServeOpts{ProviderFunc: func() *schema.Provider {
                return vmmanager6.Provider()
        }, Debug: debugMode}

        plugin.Serve(opts)
}
