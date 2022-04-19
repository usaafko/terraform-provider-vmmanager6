package vmmanager6

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
        "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
//        "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// using a global variable here so that we have an internally accessible
// way to look into our own resource definition. Useful for dynamically doing typecasts
// so that we can print (debug) our ResourceData constructs
var thisResource *schema.Resource

func resourceVmQemu() *schema.Resource {
        thisResource = &schema.Resource{
                Create:        resourceVmQemuCreate,
                Read:          resourceVmQemuRead,
                UpdateContext: resourceVmQemuUpdate,
                Delete:        resourceVmQemuDelete,
                Importer: &schema.ResourceImporter{
                        StateContext: schema.ImportStatePassthroughContext,
                },
		Schema: map[string]*schema.Schema{
                        "vmid": {
                                Type:             schema.TypeInt,
                                Optional:         true,
                                Computed:         true,
                                ForceNew:         true,
                                Description:      "The VM identifier in VMmanager (100-999999999)",
                        },
		},
		}
        return thisResource
}

func resourceVmQemuCreate(d *schema.ResourceData, meta interface{}) error {
        // create a logger for this function
	return nil
}

func resourceVmQemuUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVmQemuDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

