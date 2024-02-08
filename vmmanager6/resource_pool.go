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
var poolResource *schema.Resource

func resourcePool() *schema.Resource {
	poolResource = &schema.Resource{
		Create:        resourcePoolCreate,
		Read:          resourcePoolRead,
		UpdateContext: resourcePoolUpdate,
		Delete:        resourcePoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"pool": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Ipv4 or Ipv6 Network in CIDR format",
			},
			"ranges": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Range of ips in pool. Format: 192.168.0.1 or 192.168.0.1-192.168.0.10 or 192.168.0.0/24",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Pool Description",
				Default:     "",
			},
			"cluster": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "id of Cluster where need to enable pool",
				Default:     1,
			},
		},
	}
	return poolResource
}

func resourcePoolCreate(d *schema.ResourceData, meta interface{}) error {
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_pool_create")

	// DEBUG print out the create request
	flatValue, _ := resourceDataToFlatValues(d, poolResource)
	jsonString, _ := json.Marshal(flatValue)

	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()
	client := pconf.Client

	//check if pool exists

	vmid, err := client.GetPoolIdByName(d.Get("pool").(string))
	if err != nil {
		return err
	}
	if vmid != "0" {
		//Pool already exists
		logger.Debug().Msgf("Pool already exists id %v", vmid)
		d.SetId(vmid)
		_resourcePoolRead(d, meta)
		return nil
	}
	genRanges := d.Get("ranges").([]interface{})
	var myRanges []string
	for _, Range := range genRanges {
		myRanges = append(myRanges, Range.(string))
	}
	config := vm6api.ConfigNewPool{
		Name:    d.Get("pool").(string),
		Note:    d.Get("desc").(string),
		Ranges:  myRanges,
		Cluster: d.Get("cluster").(int),
	}
	vmid, err = config.CreatePool(client)
	if err != nil {
		return err
	}
	d.SetId(vmid)
	logger.Debug().Msgf("Finished Pool read resulting in data: '%+v'", string(jsonString))

	log.Print("[DEBUG][PoolCreate] vm creation done!")
	return nil
}

func resourcePoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)

	defer lock.unlock()
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_pool_update")

	client := pconf.Client
	logger.Info().Msg("Starting update of the pool resource")
	config, err := client.GetPoolInfo(d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	if d.HasChanges("pool", "desc") {
		err = client.UpdatePoolSettings(d.Id(), d.Get("pool").(string), d.Get("desc").(string))
		logger.Info().Msg("Change pool name and desc")
		if err != nil {
			logger.Error().Msgf("Can't update pool %v", err)
			return diag.FromErr(err)
		}
	}
	if d.HasChange("ranges") {
		oldValuesRaw, newValuesRaw := d.GetChange("ranges")
		oldValues := oldValuesRaw.([]interface{})
		newValues := newValuesRaw.([]interface{})
		for i := range oldValues {
			if !InterfaceStringsContains(newValues, oldValues[i]) { //remove something
				curRanges := config["ipnets"].([]interface{})
				for _, v := range curRanges {
					testRange := v.(map[string]interface{})["name"].(string)
					if oldValues[i].(string) == testRange {
						logger.Debug().Msgf("Delete range from pool %v", testRange)
						err = client.DeletePoolRange(int(v.(map[string]interface{})["id"].(float64)))
						if err != nil {
							return diag.FromErr(err)
						}
					}
				}

			}
		}
		for i := range newValues {
			if !InterfaceStringsContains(oldValues, newValues[i]) { //that's new value
				logger.Debug().Msgf("Add range to pool %v", newValues[i].(string))
				err = client.CreatePoolRange(d.Id(), newValues[i].(string))
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}

	}
	logger.Info().Msg("End of update of the pool resource")
	return nil
}

func resourcePoolRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	return _resourcePoolRead(d, meta)
}

func resourcePoolDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	client := pconf.Client
	err := client.DeletePool(d.Id())
	return err

}

func _resourcePoolRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	client := pconf.Client
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_pool_read")

	// Try to get information on the Pool. If this call err's out
	// that indicates the Pool does not exist. We indicate that to terraform
	// by calling a SetId("")
	_, err := client.GetPoolInfo(d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	config, err := vm6api.NewConfigPoolFromApi(d.Id(), client)
	if err != nil {
		return err
	}

	logger.Debug().Msgf("[READ] Received Pool Config from VMmanager6 API: %+v", config)

	d.Set("pool", config.Name)
	d.Set("desc", config.Note)
	var getRanges []string
	for _, val := range config.Ranges {
		getRanges = append(getRanges, val.Range)
	}
	d.Set("ranges", getRanges)
	// DEBUG print out the read result
	flatValue, _ := resourceDataToFlatValues(d, poolResource)
	jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))

	return nil
}
