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
var vxlanResource *schema.Resource

func resourceVxlan() *schema.Resource {
	vxlanResource = &schema.Resource{
		Create:        resourceVxlanCreate,
		Read:          resourceVxlanRead,
		UpdateContext: resourceVxlanUpdate,
		Delete:        resourceVxlanDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Name of VxLAN",
			},
			"account": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Account for VxLAN",
			},
			"clusters": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Description: "Array of clusters id, where this VxLAN will work",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "VxLAN Description",
				Default:     "",
			},
			"ipnets": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of networks, that need to be added to VxLAN",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Description: "Id of network inside VxLAN",
							Computed:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "IPv4 range in CIDR format",
						},
						"gateway": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Gateway for this network",
						},
					},
				},
			},
			"ippool": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "id of Ip pool, can be used for vm creation",
			},
		},
	}
	return vxlanResource
}

func resourceVxlanCreate(d *schema.ResourceData, meta interface{}) error {
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_vxlan_create")

	// DEBUG print out the create request
	flatValue, _ := resourceDataToFlatValues(d, vxlanResource)
	jsonString, _ := json.Marshal(flatValue)

	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()
	client := pconf.Client

	//check if vxlan exists

	vmid, err := client.GetVxLANIdByName(d.Get("account").(int), d.Get("name").(string))
	if err != nil {
		return err
	}
	if vmid != "0" {
		//VxLAN already exists
		logger.Debug().Msgf("VxLAN already exists id %v", vmid)
		d.SetId(vmid)
		_resourceVxlanRead(d, meta)
		return nil
	}

	ipnets := d.Get("ipnets").([]interface{})
	var NewIpnets []vm6api.VxLANipnets
	for _, net := range ipnets {
		var NewIpnet vm6api.VxLANipnets
		mynet := net.(map[string]interface{})
		NewIpnet.Name = mynet["name"].(string)
		NewIpnet.Gateway = mynet["gateway"].(string)
		NewIpnets = append(NewIpnets, NewIpnet)
	}
	var clusters []int
	resClusters := d.Get("clusters").([]interface{})
	for _, resCluster := range resClusters {
		cluster := resCluster.(int)
		clusters = append(clusters, cluster)
	}
	config := vm6api.ConfigNewVxLAN{
		Name:     d.Get("name").(string),
		Comment:  d.Get("comment").(string),
		Account:  d.Get("account").(int),
		Clusters: clusters,
		Ips:      NewIpnets,
	}

	vmid, err = config.CreateVxLAN(client)
	if err != nil {
		return err
	}
	d.SetId(vmid)

	// Update IP pool info
	err = _resourceVxlanRead(d, meta)
	if err != nil {
		return err
	}

	logger.Debug().Msgf("Finished VxLAN read resulting in data: '%+v'", string(jsonString))

	log.Print("[DEBUG][VxLANCreate] creation done!")
	return nil
}

func resourceVxlanUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Update vxlan settings
	return nil
}

func resourceVxlanRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	return _resourceVxlanRead(d, meta)
}

func resourceVxlanDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	client := pconf.Client
	err := client.DeleteVxLAN(d.Id())
	return err

}

func _resourceVxlanRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	client := pconf.Client
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_vxlan_read")

	_, err := client.GetVxLANInfo(d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	config, err := vm6api.NewConfigVxLANFromApi(d.Id(), client)
	if err != nil {
		return err
	}

	logger.Debug().Msgf("[READ] Received VxLAN Config from VMmanager6 API: %+v", config)

	d.Set("name", config.Name)
	d.Set("comment", config.Comment)
	d.Set("ippool", config.Ippool)
	d.Set("account", config.Account.Id)
	// TODO check clusters
	flatIpConfig := make([]map[string]interface{}, 0, 1)
	for _, thisip := range config.Ips {
		thisFlattenedIp := make(map[string]interface{})
		thisFlattenedIp["id"] = thisip.Id
		thisFlattenedIp["name"] = thisip.Name
		thisFlattenedIp["gateway"] = thisip.Gateway
		flatIpConfig = append(flatIpConfig, thisFlattenedIp)
	}
	if d.Set("ipnets", flatIpConfig); err != nil {
		return err
	}

	// DEBUG print out the read result
	flatValue, _ := resourceDataToFlatValues(d, vxlanResource)
	jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))

	return nil
}
