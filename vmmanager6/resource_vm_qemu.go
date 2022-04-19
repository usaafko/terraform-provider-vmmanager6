package vmmanager6

import (
	"context"
	"strings"
	"fmt"
	"log"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
        "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	vm6api "github.com/usaafko/vmmanager6-api-go"
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
                                Description:      "The VM identifier in VMmanager",
                        },
			"name": {
				Type:        schema.TypeString,
                                Optional:    true,
                                Default:     "",
                                Description: "The VM name",
			},
			"desc": {
                                Type:     schema.TypeString,
                                Optional: true,
                                DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
                                        return strings.TrimSpace(old) == strings.TrimSpace(new)
                                },
                                Default:     "",
                                Description: "The VM description",
                        },
			"cores": {
                                Type:     schema.TypeInt,
                                Optional: true,
                                Default:  1,
                        },
			"memory": {
                                Type:     schema.TypeInt,
                                Optional: true,
                                Default:  512,
                        },
			"disk": {
                                Type:     schema.TypeInt,
                                Optional: true,
                                Default:  6000,
                        },

		},
		}
        return thisResource
}

func resourceVmQemuCreate(d *schema.ResourceData, meta interface{}) error {
	// create a logger for this function
        logger, _ := CreateSubLogger("resource_vm_create")

	// DEBUG print out the create request
        flatValue, _ := resourceDataToFlatValues(d, thisResource)
        jsonString, _ := json.Marshal(flatValue)
        logger.Debug().Str("vmid", d.Id()).Msgf("Invoking VM create with resource data:  '%+v'", string(jsonString))

	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)
        //defer lock.unlock()
        client := pconf.Client
        vmName := d.Get("name").(string)

	config := vm6api.ConfigQemu{
                Name:         vmName,
                Description:  d.Get("desc").(string),
		Memory:       d.Get("memory").(int),
		QemuCores:    d.Get("cores").(int),
		QemuDisks:    d.Get("disk").(int),
	}

	log.Print("[DEBUG][QemuVmCreate] vm creation done!")
        lock.unlock()
        return resourceVmQemuRead(d, meta)
}

func resourceVmQemuUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	return _resourceVmQemuRead(d, meta)
}

func resourceVmQemuDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func _resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)
        defer lock.unlock()
        client := pconf.Client
        // create a logger for this function
        logger, _ := CreateSubLogger("resource_vm_read")

	_, _, vmID, err := parseResourceId(d.Id())
        if err != nil {
                d.SetId("")
                return fmt.Errorf("unexpected error when trying to read and parse the resource: %v", err)
        }

        logger.Info().Int("vmid", vmID).Msg("Reading configuration for vmid")
        vmr := vm6api.NewVmRef(vmID)

	// Try to get information on the vm. If this call err's out
        // that indicates the VM does not exist. We indicate that to terraform
        // by calling a SetId("")
        _, err = client.GetVmInfo(vmr)
	if err != nil {
                d.SetId("")
                return nil
        }
        config, err := vm6api.NewConfigQemuFromApi(vmr, client)
        if err != nil {
                return err
        }

	vmState, err := client.GetVmState(vmr)
        log.Printf("[DEBUG] VM status: %s", vmState["status"])

	if err == nil && vmState["status"] == "started" {
                log.Printf("[DEBUG] VM is running, cheking the IP")
	}

	if err != nil {
                return err
        }

        logger.Debug().Int("vmid", vmID).Msgf("[READ] Received Config from VMmanager6 API: %+v", config)

	d.SetId(resourceId(vmr.Node(), "qemu", vmr.VmId()))
	d.Set("name", config.Name)
	d.Set("desc", config.Description)
	d.Set("memory", config.Memory)
	d.Set("cores", config.QemuCores)
	d.Set("disk", config.QemuDisks)

	// DEBUG print out the read result
        flatValue, _ := resourceDataToFlatValues(d, thisResource)
        jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Int("vmid", vmID).Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))

        return nil
}

