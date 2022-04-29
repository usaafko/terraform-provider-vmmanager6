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
	vm6api "github.com/usaafko/vmmanager6-api-go"
//        "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// using a global variable here so that we have an internally accessible
// way to look into our own resource definition. Useful for dynamically doing typecasts
// so that we can print (debug) our ResourceData constructs
var accountResource *schema.Resource

func resourceAccount() *schema.Resource {
        accountResource = &schema.Resource{
                Create:        resourceAccountCreate,
                Read:          resourceAccountRead,
                UpdateContext: resourceAccountUpdate,
                Delete:        resourceAccountDelete,
                Importer: &schema.ResourceImporter{
                        StateContext: schema.ImportStatePassthroughContext,
                },
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
                                Required:    true,
                                Description: "User log in to VMmanager by email",
                                ForceNew: true,
			},
			"state": {
                                Type:     schema.TypeString,
                                Computed: true,
                                Description: "Internal - user state",
                        },
			"role": {
                                Type:     schema.TypeString,
                                Optional: true,
                                Description: "User role, must be @admin or @advanced_user or @user",           
                                Default: "@admin",
                                ValidateFunc: validation.StringInSlice([]string{
					"@admin",
					"@advanced_user",
					"@user",
				}, false),
                        },
                        "password": {
                                Type:     schema.TypeString,
                                Required: true,
                                Sensitive: true,
                                Description: "User password",
                        },
		},
		}
        return accountResource
}

func resourceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	// create a logger for this function
        logger, _ := CreateSubLogger("resource_account_create")

	// DEBUG print out the create request
        flatValue, _ := resourceDataToFlatValues(d, accountResource)
        jsonString, _ := json.Marshal(flatValue)

	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)
        //defer lock.unlock()
        client := pconf.Client

	//check if account exists
	
	vmid, err := client.GetAccountIdByEmail(d.Get("email").(string))
	if err != nil {
		return err
	}
	if vmid != "0" {
		//Account already exists
		logger.Debug().Msgf("Account already exists id %v", vmid)
		d.SetId(vmid)
		_resourceAccountRead(d, meta)
		return nil
	}

	config := vm6api.ConfigNewAccount{
                Email:		d.Get("email").(string),
                Role:      	d.Get("role").(string),
                Password:       d.Get("password").(string),
	}
	vmid, err = config.CreateAccount(client)
	if err != nil {
		return err
	}
	d.SetId(vmid)
	logger.Debug().Msgf("Finished account read resulting in data: '%+v'", string(jsonString))
	
	log.Print("[DEBUG][AccountCreate] creation done!")
        lock.unlock()
        return nil
}

func resourceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)

        defer lock.unlock()
	// create a logger for this function
        logger, _ := CreateSubLogger("resource_account_update")

	client := pconf.Client
	logger.Info().Msg("Starting update of the account resource")

	_, err := client.GetAccountInfo(d.Id())
	if err != nil {
                d.SetId("")
                return nil
        }

        if d.HasChange("role"){
		//TODO: change user role
        }
        logger.Info().Msg("End of update of the account resource")
	return nil
}

func resourceAccountRead(d *schema.ResourceData, meta interface{}) error {
	return _resourceAccountRead(d, meta)
}

func resourceAccountDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)
        defer lock.unlock()

        client := pconf.Client
	err := client.DeleteAccount(d.Id())
	return err

}

func _resourceAccountRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
        lock := pmParallelBegin(pconf)
        defer lock.unlock()
        client := pconf.Client
        // create a logger for this function
        logger, _ := CreateSubLogger("resource_account_read")

	_, err := client.GetAccountInfo(d.Id())
	if err != nil {
                d.SetId("")
                return nil
        }
        config, err := vm6api.NewConfigAccountFromApi(d.Id(), client)
        if err != nil {
                return err
        }


        logger.Debug().Msgf("[READ] Received Account Config from VMmanager6 API: %+v", config)

	d.Set("state") = config.State
	d.Set("role") = config.Role
	
	// DEBUG print out the read result
        flatValue, _ := resourceDataToFlatValues(d, accountResource)
        jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Msgf("Finished account read resulting in data: '%+v'", string(jsonString))

        return nil
}

