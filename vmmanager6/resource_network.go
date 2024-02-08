package vmmanager6

import (
	"context"
	//	"strings"
	"log"
	//	"strconv"
	//	"fmt"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	vm6api "github.com/naughtyerica/vmmanager6-api-go"
	// "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// using a global variable here so that we have an internally accessible
// way to look into our own resource definition. Useful for dynamically doing typecasts
// so that we can print (debug) our ResourceData constructs
var networkResource *schema.Resource

func resourceNetwork() *schema.Resource {
	networkResource = &schema.Resource{
		Create:        resourceNetworkCreate,
		Read:          resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		Delete:        resourceNetworkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Ipv4 or Ipv6 Network in CIDR format",
				ForceNew:    true,
			},
			"gateway": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Ip address of gateway",
				ForceNew:    true,
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network Description",
				Default:     "",
			},
		},
	}
	return networkResource
}

func resourceNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_network_create")

	// DEBUG print out the create request
	flatValue, _ := resourceDataToFlatValues(d, networkResource)
	jsonString, _ := json.Marshal(flatValue)

	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	//defer lock.unlock()
	client := pconf.Client

	//check if network exists

	vmid, err := client.GetNetworkIdByName(d.Get("network").(string))
	if err != nil {
		return err
	}
	if vmid != "0" {
		//Network already exists
		logger.Debug().Msgf("Network already exists id %v", vmid)
		d.SetId(vmid)
		_resourceNetworkRead(d, meta)
		return nil
	}

	config := vm6api.ConfigNewNetwork{
		Name:    d.Get("network").(string),
		Gateway: d.Get("gateway").(string),
		Note:    d.Get("desc").(string),
	}
	vmid, err = config.CreateNetwork(client)
	if err != nil {
		return err
	}
	d.SetId(vmid)
	logger.Debug().Msgf("Finished network read resulting in data: '%+v'", string(jsonString))

	log.Print("[DEBUG][NetworkCreate] vm creation done!")
	lock.unlock()
	return nil
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)

	defer lock.unlock()
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_network_update")

	client := pconf.Client
	logger.Info().Msg("Starting update of the network resource")

	_, err := client.GetNetworkInfo(d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	if d.HasChange("desc") {
		err = client.UpdateNetworkDescription(d.Id(), d.Get("desc").(string))
		logger.Info().Msg("Change network desc")
		if err != nil {
			logger.Error().Msgf("Can't update network %v", err)
			return diag.FromErr(err)
		}
	}
	logger.Info().Msg("End of update of the network resource")
	return nil
}

func resourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	return _resourceNetworkRead(d, meta)
}

func resourceNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	client := pconf.Client
	err := client.DeleteNetwork(d.Id())
	return err

}

func _resourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	client := pconf.Client
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_network_read")

	// Try to get information on the network. If this call err's out
	// that indicates the network does not exist. We indicate that to terraform
	// by calling a SetId("")
	_, err := client.GetNetworkInfo(d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}
	config, err := vm6api.NewConfigNetworkFromApi(d.Id(), client)
	if err != nil {
		return err
	}

	logger.Debug().Msgf("[READ] Received Network Config from VMmanager6 API: %+v", config)

	d.Set("network", config.Name)
	d.Set("gateway", config.Gateway)
	d.Set("desc", config.Note)

	// DEBUG print out the read result
	flatValue, _ := resourceDataToFlatValues(d, networkResource)
	jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))

	return nil
}
