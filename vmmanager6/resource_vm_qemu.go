package vmmanager6

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	vm6api "github.com/usaafko/vmmanager6-api-go"
	// "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Number of vCPU's for VM",
			},
			"memory": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     512,
				Description: "RAM Size of VM in Megabytes",
			},
			"disk": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     6000,
				Description: "Disk Size of VM in Megabytes",
			},
			"disks": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:	 true,
				Description: "Secondary Disk of VM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size_mib": {
							Type:        schema.TypeInt,
							Description: "Disk Size of VM in Megabytes",
							Optional:    true,
							Default:     6000,
						},
						"boot_order": {
							Type:        schema.TypeInt,
							Description: "Boot order of the Disk",
							Required:	 true,
						},
						"tags": {
							Type:        schema.TypeList,
							Description: "Tag of the Disk ",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeInt,
							},
						},
					},
				},
			},
			"preset": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "id of VM preset. Preset will overwrite your cpu/mem/disk settings",
			},
			"disk_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Internal variable. Main disk ID of VM",
			},
			"cluster": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				ForceNew:    true,
				Description: "VMmanager 6 cluster id",
			},
			"node": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default: 	 0,
				ForceNew:    true,
				Default:     0,
				Description: "VMmanager 6 node id",
			},
			"account": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     3,
				Description: "VMmanager user id",
			},
			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Domain for VM's ip addresses and hostname",
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return new == "**********"
				},
				Description: "Password for VM",
			},
			"os": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "VMmanager 6 template id",
			},
			"cpu_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cpu mode. Can be default, host-model, host-passthrough",
				Default:     "default",
				ValidateFunc: validation.StringInSlice([]string{
					"default",
					"host-model",
					"host-passthrough",
				}, false),
			},
			"anti_spoofing": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Anti spoofing",
			},
			"ipv4_number": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of ipv4 addresses",
			},
			"ipv4_pools": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "VMmanager ip pools, to use for ip assignment",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"custom_interfaces": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "You can set some ip address manually (use ip_name) or using pool id (ip_pool)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bridge": {
							Type:        schema.TypeString,
							Description: "Bridge name for interface",
							Optional:    true,
							Default:     "vmbr0",
						},
						"ip_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Ip address to apply",
						},
						"ippool": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Pool of ip addresses to apply",
						},
						"ip_count": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "How many ips add to this interface from ip_pool",
							Default:     1,
						},
					},
				},
			},
			"vxlan": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "Use vxlan to create VM in local network without public ips, or mix it with custom_interfaces",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "id of VxLAN",
						},
						"ipnet": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "id of network inside VxLAN",
						},
						"ipv4_number": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "How many ips from VxLAN needed",
							Default:     1,
						},
					},
				},
			},
			"ip_addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Internal. List of vms ip addresses",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"domain": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"family": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"netid": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"gateway": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"addr": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mask": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"recipes": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "Array of recipes and params",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"recipe": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "id of recipe",
							ForceNew:    true,
						},
						"recipe_params": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Array of recipe params",
							ForceNew:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "param name",
										ForceNew:    true,
									},
									"value": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "param value",
										ForceNew:    true,
									},
								},
							},
						},
					},
				},
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

	// Collect pools from config
	ipv4_pools := d.Get("ipv4_pools").([]interface{})
	var ipv4_pools_int []int
	for _, ippool := range ipv4_pools {
		ipv4_pools_int = append(ipv4_pools_int, ippool.(int))
	}

	// Collect recipes from config
	recipes_config := d.Get("recipes").([]interface{})
	var recipes_api []vm6api.RecipeConfig

	j, err := json.Marshal(recipes_config)
	err = json.Unmarshal(j, &recipes_api)
	if err != nil {
		return err
	}
	
	// Collect disks from config
	secondary_disk_config := d.Get("disks").([]interface{})
	var secondary_disk_api []vm6api.ConfigSecondaryDisk

	j, err = json.Marshal(secondary_disk_config)
	err = json.Unmarshal(j, &secondary_disk_api)
	if err != nil {
		return err
	}

	config := vm6api.ConfigNewQemu{
		Name:             	d.Get("name").(string),
		Description:      	d.Get("desc").(string),
		Memory:           	d.Get("memory").(int),
		QemuCores:        	d.Get("cores").(int),
		QemuDisks:        	d.Get("disk").(int),
		QemuSecondaryDisks: secondary_disk_api,
		Cluster:          	d.Get("cluster").(int),
		Node:             	d.Get("node").(int),
		Account:          	d.Get("account").(int),
		Domain:           	d.Get("domain").(string),
		Password:         	d.Get("password").(string),
		IPv4:             	d.Get("ipv4_number").(int),
		Os:               	d.Get("os").(int),
		Anti_spoofing:    	d.Get("anti_spoofing").(bool),
		CpuMode:          	d.Get("cpu_mode").(string),
		Preset:           	d.Get("preset").(int),
		IPv4Pools:        	ipv4_pools_int,
		Recipes:          	recipes_api,
		CustomInterfaces: 	d.Get("custom_interfaces").([]interface{}),
		Vxlans:           	d.Get("vxlan").([]interface{}),
	}
	vmid, err := config.CreateVm(client)
	if err != nil {
		return err
	}
	d.SetId(fmt.Sprint(vmid))

	logger.Debug().Int("vmid", vmid).Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))
	err = _resourceVmQemuRead(d, meta)
	if err != nil {
		return err
	}

	log.Print("[DEBUG][QemuVmCreate] vm creation done!")
	lock.unlock()
	return nil
}

func resourceVmQemuUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	// create a logger for this function
	logger, _ := CreateSubLogger("resource_vm_update")

	client := pconf.Client
	vmID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	vmr := vm6api.NewVmRef(vmID)
	logger.Info().Int("vmid", vmID).Msg("Starting update of the VM resource")

	// Try to get information on the vm. If this call err's out
	// that indicates the VM does not exist.
	_, err = client.GetVmInfo(vmr)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("disk") {
		oldValuesRaw, newValuesRaw := d.GetChange("disk")
		oldValues := oldValuesRaw.(int)
		newValues := newValuesRaw.(int)
		if oldValues > newValues {
			return diag.Errorf("Can't shrink VM's disk")
		}
	}

	// VMmanager has different APIs to change things.
	// 1. Resources
	if d.HasChanges("cores", "memory", "cpu_mode") {
		config := vm6api.ResourcesQemu{
			Cores:   d.Get("cores").(int),
			Memory:  d.Get("memory").(int),
			CpuMode: d.Get("cpu_mode").(string),
		}
		logger.Debug().Int("vmid", vmID).Msgf("Updating VM with the following configuration: %+v", config)
		err = config.UpdateResources(vmr, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	// 2. VM settings
	if d.HasChanges("name", "desc") {
		config := vm6api.UpdateConfigQemu{
			Name:        d.Get("name").(string),
			Description: d.Get("desc").(string),
		}
		logger.Debug().Int("vmid", vmID).Msgf("Updating VM with the following configuration: %+v", config)
		err = config.UpdateConfig(vmr, client)
		if err != nil {
			return diag.FromErr(err)
		}

	}
	// 3. Change OS
	if d.HasChange("os") {
		config := vm6api.ReinstallOS{
			Id:        d.Get("os").(int),
			Password:  d.Get("password").(string),
			EmailMode: "saas_only",
		}
		logger.Debug().Int("vmid", vmID).Msgf("Updating VM with the following configuration: %+v", config)
		err = config.ReinstallOS(vmr, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	// 4. Change password
	if d.HasChange("password") {
		logger.Debug().Int("vmid", vmID).Msgf("Updating VM password")
		err = client.ChangePassword(vmr, d.Get("password").(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	// 5. Owner
	if d.HasChange("account") {
		logger.Debug().Int("vmid", vmID).Msgf("Updating VM owner %v", d.Get("account").(int))
		err = client.ChangeOwner(vmr, d.Get("account").(int))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	// 6. Disk
	if d.HasChange("disk") {
		config := vm6api.ConfigDisk{
			Size: d.Get("disk").(int),
			Id:   d.Get("disk_id").(int),
		}
		logger.Debug().Int("vmid", vmID).Msgf("Updating VM with the following configuration: %+v", config)
		err = config.UpdateDisk(client)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	// 7. Domain
	if d.HasChange("domain") {
		vmIps := d.Get("ip_addresses").([]interface{})
		flatIpConfig := make([]map[string]interface{}, 0, 1)
		newdomain := d.Get("domain").(string)
		for _, vmIp := range vmIps {
			thisIp := vmIp.(map[string]interface{})
			vmIpId := thisIp["id"].(int)
			// change ptr for that ip
			err = client.UpdatePtr(vmIpId, newdomain)
			if err != nil {
				return diag.FromErr(err)
			}
			thisIp["domain"] = newdomain
			flatIpConfig = append(flatIpConfig, thisIp)
		}
		d.Set("ip_addresses", flatIpConfig)
	}
	var diags diag.Diagnostics
	lock.unlock()
	return diags
}

func resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	return _resourceVmQemuRead(d, meta)
}

func resourceVmQemuDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	client := pconf.Client
	vmID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	vmr := vm6api.NewVmRef(vmID)
	err = client.DeleteQemuVm(vmr)
	return err

}

func _resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
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
	d.Set("disk", config.QemuDisks.Size)
	d.Set("cluster", config.Cluster.Id)
	d.Set("node", config.Node.Id)
	d.Set("account", config.Account.Id)
	d.Set("domain", config.Domain)
	d.Set("os", config.Os.Id)
	d.Set("disk_id", config.QemuDisks.Id)

	ipconfig, err := vm6api.NewConfigQemuIpsFromApi(vmr, client)
	if err != nil {
		return err
	}

	flatIpConfig := make([]map[string]interface{}, 0, 1)
	for _, thisip := range ipconfig {
		thisFlattenedIp := make(map[string]interface{})
		thisFlattenedIp["domain"] = thisip.Domain
		thisFlattenedIp["family"] = thisip.Family
		thisFlattenedIp["id"] = thisip.Id
		thisFlattenedIp["netid"] = thisip.NetId
		thisFlattenedIp["gateway"] = thisip.Gateway
		thisFlattenedIp["addr"] = thisip.Addr
		thisFlattenedIp["mask"] = thisip.Mask
		flatIpConfig = append(flatIpConfig, thisFlattenedIp)
	}
	if d.Set("ip_addresses", flatIpConfig); err != nil {
		return err
	}

	// DEBUG print out the read result
	flatValue, _ := resourceDataToFlatValues(d, thisResource)
	jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Int("vmid", vmID).Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))

	return nil
}
