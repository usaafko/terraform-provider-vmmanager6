package vmmanager6

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	vm6api "github.com/naughtyerica/vmmanager6-api-go"
)

type providerConfiguration struct {
	Client                             *vm6api.Client
	MaxParallel                        int
	CurrentParallel                    int
	MaxVMID                            int
	Mutex                              *sync.Mutex
	Cond                               *sync.Cond
	LogFile                            string
	LogLevels                          map[string]string
	DangerouslyIgnoreUnknownAttributes bool
}

// Provider - Terrafrom properties for vmmanager6
func Provider() *schema.Provider {
	return &schema.Provider{

		Schema: map[string]*schema.Schema{
			"pm_email": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_EMAIL", nil),
				Description: "Email e.g. admin@example.com",
			},
			"pm_password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_PASS", nil),
				Description: "Password to authenticate into vmmanager6",
				Sensitive:   true,
			},
			"pm_api_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_API_URL", nil),
				Description: "https://host.fqdn/vm/v3",
			},
			"pm_api_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				DefaultFunc: schema.EnvDefaultFunc("PM_API_TOKEN", nil),
				Description: "API Token",
			},
			"pm_parallel": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  4,
			},
			"pm_tls_insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_TLS_INSECURE", true), //we assume it's a lab!
				Description: "By default, every TLS connection is verified to be secure. This option allows terraform to proceed and operate on servers considered insecure. For example if you're connecting to a remote host and you do not have the CA cert that issued the VMmanager 6 api url's certificate.",
			},
			"pm_log_enable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable provider logging to get VMmanager API logs",
			},
			"pm_log_levels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Configure the logging level to display; trace, debug, info, warn, etc",
			},
			"pm_log_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "terraform-plugin-vmmanager6.log",
				Description: "Write logs to this specific file",
			},
			"pm_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_TIMEOUT", defaultTimeout),
				Description: "How much second to wait for operations for both provider and api-client, default is 300s",
			},
			"pm_dangerously_ignore_unknown_attributes": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_DANGEROUSLY_IGNORE_UNKNOWN_ATTRIBUTES", false),
				Description: "By default this provider will exit if an unknown attribute is found. This is to prevent the accidential destruction of VMs or Data when something in the VMmanager 6 API has changed/updated and is not confirmed to work with this provider. Set this to true at your own risk. It may allow you to proceed in cases when the provider refuses to work, but be aware of the danger in doing so.",
			},
			"pm_debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PM_DEBUG", false),
				Description: "Enable or disable the verbose debug output from VMmanager 6 api",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"vmmanager6_vm_qemu": resourceVmQemu(),
			"vmmanager6_network": resourceNetwork(),
			"vmmanager6_pool":    resourcePool(),
			"vmmanager6_account": resourceAccount(),
			"vmmanager6_vxlan":   resourceVxlan(),
			//        "vmmanager6_lxc":      resourceLxc(),
			//        "vmmanager6_lxc_disk": resourceLxcDisk(),
			//        "vmmanager6_pool":     resourcePool(),
		},

		ConfigureFunc: providerConfigure,
	}
}
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	client, err := getClient(
		d.Get("pm_api_url").(string),
		d.Get("pm_email").(string),
		d.Get("pm_password").(string),
		d.Get("pm_api_token").(string),
		d.Get("pm_tls_insecure").(bool),
		d.Get("pm_timeout").(int),
		d.Get("pm_debug").(bool),
	)
	if err != nil {
		return nil, err
	}

	// look to see what logging we should be outputting according to the provider configuration
	logLevels := make(map[string]string)
	for logger, level := range d.Get("pm_log_levels").(map[string]interface{}) {
		levelAsString, ok := level.(string)
		if ok {
			logLevels[logger] = levelAsString
		} else {
			return nil, fmt.Errorf("invalid logging level %v for %v. Be sure to use a string", level, logger)
		}
	}

	// actually configure logging
	// note that if enable is false here, the configuration will squash all output
	ConfigureLogger(
		d.Get("pm_log_enable").(bool),
		d.Get("pm_log_file").(string),
		logLevels,
	)

	var mut sync.Mutex
	return &providerConfiguration{
		Client:                             client,
		MaxParallel:                        d.Get("pm_parallel").(int),
		CurrentParallel:                    0,
		MaxVMID:                            -1,
		Mutex:                              &mut,
		Cond:                               sync.NewCond(&mut),
		LogFile:                            d.Get("pm_log_file").(string),
		LogLevels:                          logLevels,
		DangerouslyIgnoreUnknownAttributes: d.Get("pm_dangerously_ignore_unknown_attributes").(bool),
	}, nil
}
func getClient(pm_api_url string,
	pm_email string,
	pm_password string,
	pm_api_token string,
	pm_tls_insecure bool,
	pm_timeout int,
	pm_debug bool) (*vm6api.Client, error) {

	tlsconf := &tls.Config{InsecureSkipVerify: true}
	if !pm_tls_insecure {
		tlsconf = nil
	}

	var err error

	if pm_password != "" && pm_api_token != "" {
		err = fmt.Errorf("password and API token both exist, choose one or the other")
	}
	if pm_password == "" && pm_api_token == "" {
		err = fmt.Errorf("password and API token do not exist, one of these must exist")
	}

	client, _ := vm6api.NewClient(pm_api_url, nil, tlsconf, pm_timeout)
	*vm6api.Debug = pm_debug

	// User+Pass authentication
	if pm_email != "" && pm_password != "" {
		err = client.Login(pm_email, pm_password)
	}

	// API authentication
	if pm_api_token != "" {
		client.SetAPIToken(pm_api_token)
	}

	if err != nil {
		return nil, err
	}
	return client, nil
}

type pmApiLockHolder struct {
	locked bool
	pconf  *providerConfiguration
}

func (lock *pmApiLockHolder) lock() {
	if lock.locked {
		return
	}
	lock.locked = true
	pconf := lock.pconf
	pconf.Mutex.Lock()
	for pconf.CurrentParallel >= pconf.MaxParallel {
		pconf.Cond.Wait()
	}
	pconf.CurrentParallel++
	pconf.Mutex.Unlock()
}

func (lock *pmApiLockHolder) unlock() {
	if !lock.locked {
		return
	}
	lock.locked = false
	pconf := lock.pconf
	pconf.Mutex.Lock()
	pconf.CurrentParallel--
	pconf.Cond.Signal()
	pconf.Mutex.Unlock()
}

func pmParallelBegin(pconf *providerConfiguration) *pmApiLockHolder {
	lock := &pmApiLockHolder{
		pconf:  pconf,
		locked: false,
	}
	lock.lock()
	return lock
}

func resourceId(targetNode string, resType string, vmId int) string {
	return fmt.Sprintf("%s/%s/%d", targetNode, resType, vmId)
}

func parseResourceId(resId string) (targetNode string, resType string, vmId int, err error) {
	if !rxRsId.MatchString(resId) {
		return "", "", -1, fmt.Errorf("invalid resource format: %s. Must be node/type/vmId", resId)
	}
	idMatch := rxRsId.FindStringSubmatch(resId)
	targetNode = idMatch[1]
	resType = idMatch[2]
	vmId, err = strconv.Atoi(idMatch[3])
	return
}

func clusterResourceId(resType string, resId string) string {
	return fmt.Sprintf("%s/%s", resType, resId)
}

func parseClusterResourceId(resId string) (resType string, id string, err error) {
	if !rxClusterRsId.MatchString(resId) {
		return "", "", fmt.Errorf("invalid resource format: %s. Must be type/resId", resId)
	}
	idMatch := rxClusterRsId.FindStringSubmatch(resId)
	return idMatch[1], idMatch[2], nil
}
