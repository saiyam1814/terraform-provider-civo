package civo

import (
	"context"
	"log"

	"github.com/civo/civogo"
	"github.com/civo/terraform-provider-civo/internal/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Firewall resource with this we can create and manage all firewall
func resourceFirewall() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Civo firewall resource. This can be used to create, modify, and delete firewalls.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: utils.ValidateName,
				Description:  "The firewall name",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The firewall region, if is not defined we use the global defined in the provider",
			},
			"create_default_rules": {
				Type:        schema.TypeBool,
				Default:     true,
				Optional:    true,
				ForceNew:    true,
				Description: "The create rules flag is used to create the default firewall rules, if is not defined will be set to true",
			},
			// As the backend has no support for updating network ID we replace it if the
			// network_id changes
			"network_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The firewall network, if is not defined we use the default network",
			},
		},
		CreateContext: resourceFirewallCreate,
		ReadContext:   resourceFirewallRead,
		UpdateContext: resourceFirewallUpdate,
		DeleteContext: resourceFirewallDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// function to create a firewall
func resourceFirewallCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*civogo.Client)
	var networkID string
	var CreateRules bool

	// overwrite the region if it's defined
	if region, ok := d.GetOk("region"); ok {
		apiClient.Region = region.(string)
	}

	if attr, ok := d.GetOk("network_id"); ok {
		networkID = attr.(string)
	} else {
		network, err := apiClient.GetDefaultNetwork()
		if err != nil {
			return diag.Errorf("[ERR] failed to get the default network: %s", err)
		}
		networkID = network.ID
	}

	CreateRules = d.Get("create_default_rules").(bool)

	log.Printf("[INFO] creating a new firewall %s", d.Get("name").(string))

	firewall, err := apiClient.NewFirewall(d.Get("name").(string), networkID, &CreateRules)
	if err != nil {
		return diag.Errorf("[ERR] failed to create a new firewall: %s", err)
	}

	d.SetId(firewall.ID)

	return resourceFirewallRead(ctx, d, m)
}

// function to read a firewall
func resourceFirewallRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*civogo.Client)

	// overwrite the region if it's defined
	if region, ok := d.GetOk("region"); ok {
		apiClient.Region = region.(string)
	}

	log.Printf("[INFO] retriving the firewall %s", d.Id())
	resp, err := apiClient.FindFirewall(d.Id())
	if err != nil {
		if resp == nil {
			d.SetId("")
			return nil
		}

		return diag.Errorf("[ERR] error retrieving firewall: %s", err)
	}

	d.Set("name", resp.Name)
	d.Set("network_id", resp.NetworkID)

	return nil
}

// function to update the firewall
func resourceFirewallUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*civogo.Client)

	// overwrite the region if it's defined
	if region, ok := d.GetOk("region"); ok {
		apiClient.Region = region.(string)
	}

	if d.HasChange("name") {
		if d.Get("name").(string) != "" {
			firewall := civogo.FirewallConfig{
				Name: d.Get("name").(string),
			}
			log.Printf("[INFO] updating the firewall name, %s", d.Id())
			_, err := apiClient.RenameFirewall(d.Id(), &firewall)
			if err != nil {
				return diag.Errorf("[WARN] an error occurred while tring to rename the firewall %s, %s", d.Id(), err)
			}
		}
	}

	return resourceFirewallRead(ctx, d, m)
}

// function to delete a firewall
func resourceFirewallDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*civogo.Client)

	// overwrite the region if it's defined
	if region, ok := d.GetOk("region"); ok {
		apiClient.Region = region.(string)
	}

	firewallID := d.Id()
	log.Printf("[INFO] Checking if firewall %s exists", firewallID)
	_, err := apiClient.FindFirewall(firewallID)
	if err != nil {
		log.Printf("[INFO] Unable to find firewall %s - probably it's been deleted", firewallID)
		return nil
	}

	log.Printf("[INFO] deleting the firewall %s", firewallID)
	_, err = apiClient.DeleteFirewall(firewallID)
	if err != nil {
		return diag.Errorf("[ERR] an error occurred while tring to delete the firewall %s, %s", firewallID, err)
	}
	return nil
}
