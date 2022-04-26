package vmmanager6

import (
	"context"
	"strings"
	"log"
	"strconv"
	"fmt"
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
			"name": {
				Type:        schema.TypeString,
                                Required:    true,
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
                                Required: true,
                                Description: "Number of vCPU's for VM",
                        },
			"memory": {
                                Type:     schema.TypeInt,
                                Required: true,
                                Description: "RAM Size of VM in Megabytes",
                        },
			"disk": {
                                Type:     schema.TypeInt,
                                Required: true,
                                Description: "Disk Size of VM in Megabytes",
                        },
			"cluster": {
                                Type:     schema.TypeInt,
                                Optional: true,
                                Default:     1,
                                Description: "VMmanager 6 cluster id",
			},
			"account": {
                                Type:     schema.TypeInt,
                                Optional: true,
                                Default:     3,
                                Description: "VMmanager user id",
			},
			"domain": {
                                Type:     schema.TypeString,
                                Optional: true,
                                Default:     "",
                                Description: "Domain for VM's ip addresses and hostname",
			},
			"password": {
                                Type:     schema.TypeString,
                                Required:    true,
                                Description: "Password for VM",
			},
			"os": {
                                Type:     schema.TypeInt,
                                Required:    true,
                                Description: "VMmanager 6 template id",
			},
			"ipv4_number": {
				Type:     schema.TypeInt,
				Optional:    true,
				Description: "Number of ipv4 addresses",
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

	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)
        //defer lock.unlock()
        client := pconf.Client

	config := vm6api.ConfigNewQemu{
                Name:         d.Get("name").(string),
                Description:  d.Get("desc").(string),
		Memory:       d.Get("memory").(int),
		QemuCores:    d.Get("cores").(int),
		QemuDisks:    d.Get("disk").(int),
		Cluster:      d.Get("cluster").(int),
		Account:      d.Get("account").(int),
		Domain:       d.Get("domain").(string),
		Password:     d.Get("password").(string),
		IPv4:         d.Get("ipv4_number").(int),
		Os:           d.Get("os").(int),
	}
	vmid, err := config.CreateVm(client)
	if err != nil {
		return err
	}
	d.SetId(fmt.Sprint(vmid))
	logger.Debug().Int("vmid", vmid).Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))
	log.Print("[DEBUG][QemuVmCreate] vm creation done!")
        lock.unlock()
        return nil
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

	vmID, err := strconv.Atoi(d.Id())
	if err != nil {
                return err
        }
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
        log.Printf("[DEBUG] VM status: %s", vmState)

	if err == nil && vmState == "active" {
                log.Printf("[DEBUG] VM is running, cheking the IP")
	}

	if err != nil {
                return err
        }

        logger.Debug().Int("vmid", vmID).Msgf("[READ] Received Config from VMmanager6 API: %+v", config)

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

