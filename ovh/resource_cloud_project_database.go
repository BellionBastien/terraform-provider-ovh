package ovh

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/ovh/go-ovh/ovh"
	"github.com/ovh/terraform-provider-ovh/ovh/helpers"
)

func resourceCloudProjectDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudProjectDatabaseCreate,
		Read:   resourceCloudProjectDatabaseRead,
		Delete: resourceCloudProjectDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"service_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_CLOUD_PROJECT_SERVICE", nil),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"engine": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"private_network_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"private_subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"plan": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"nodes_pattern": {
				Type:     schema.TypeList,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"flavor": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"region": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"number": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			// Computed
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudProjectDatabaseImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	givenId := d.Id()
	splitId := strings.SplitN(givenId, "/", 2)
	if len(splitId) != 3 {
		return nil, fmt.Errorf("Import Id is not service_name/databaseId formatted")
	}
	serviceName := splitId[0]
	id := splitId[1]
	d.SetId(id)
	d.Set("service_name", serviceName)

	results := make([]*schema.ResourceData, 1)
	results[0] = d
	return results, nil
}

func resourceCloudProjectDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	engine := d.Get("engine").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/database/%s", serviceName, engine)
	params := (&CloudProjectDatabaseCreateOpts{}).FromResource(d)
	res := &CloudProjectDatabaseResponse{}

	log.Printf("[DEBUG] Will create Database: %+v", params)
	err := config.OVHClient.Post(endpoint, params, res)
	if err != nil {
		return fmt.Errorf("calling Post %s with params %s:\n\t %q", endpoint, params, err)
	}

	log.Printf("[DEBUG] Waiting for database %s to be READY", res.Id)
	err = waitForCloudProjectDatabaseReady(config.OVHClient, serviceName, engine, res.Id)
	if err != nil {
		return fmt.Errorf("timeout while waiting database %s to be READY: %v", res.Id, err)
	}
	log.Printf("[DEBUG] database %s is READY", res.Id)

	d.SetId(res.Id)

	return resourceCloudProjectDatabaseRead(d, meta)
}

func resourceCloudProjectDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	engine := d.Get("engine").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/database/%s/%s", serviceName, engine, d.Id())
	res := &CloudProjectDatabaseResponse{}

	log.Printf("[DEBUG] Will read database %s from project: %s", d.Id(), serviceName)
	if err := config.OVHClient.Get(endpoint, res); err != nil {
		return helpers.CheckDeleted(d, err, endpoint)
	}
	for k, v := range res.ToMap() {
		if k != "id" {
			d.Set(k, v)
		} else {
			d.SetId(fmt.Sprint(v))
		}
	}

	log.Printf("[DEBUG] Read Database %+v", res)
	return nil
}

func resourceCloudProjectDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	engine := d.Get("engine").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/database/%s/%s", serviceName, engine, d.Id())

	log.Printf("[DEBUG] Will delete database %s from project: %s", d.Id(), serviceName)
	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		return helpers.CheckDeleted(d, err, endpoint)
	}

	log.Printf("[DEBUG] Waiting for database %s to be DELETED", d.Id())
	err = waitForCloudProjectDatabaseDeleted(config.OVHClient, serviceName, engine, d.Id())
	if err != nil {
		return fmt.Errorf("timeout while waiting database %s to be DELETED: %v", d.Id(), err)
	}
	log.Printf("[DEBUG] database %s is DELETED", d.Id())

	d.SetId("")

	return nil
}

func cloudProjectDatabaseExists(serviceName, engine string, id string, client *ovh.Client) error {
	res := &CloudProjectDatabaseResponse{}

	endpoint := fmt.Sprintf("/cloud/project/%s/database/%s/%s", serviceName, engine, id)
	return client.Get(endpoint, res)
}

func waitForCloudProjectDatabaseReady(client *ovh.Client, serviceName, engine string, databaseId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"PENDING", "CREATING"},
		Target:  []string{"READY"},
		Refresh: func() (interface{}, string, error) {
			res := &CloudProjectDatabaseResponse{}
			endpoint := fmt.Sprintf("/cloud/project/%s/database/%s/%s", serviceName, engine, databaseId)
			err := client.Get(endpoint, res)
			if err != nil {
				return res, "", err
			}

			return res, res.Status, nil
		},
		Timeout:    20 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	return err
}

func waitForCloudProjectDatabaseDeleted(client *ovh.Client, serviceName, engine string, databaseId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"DELETING"},
		Target:  []string{"DELETED"},
		Refresh: func() (interface{}, string, error) {
			res := &CloudProjectDatabaseResponse{}
			endpoint := fmt.Sprintf("/cloud/project/%s/%s/%s", serviceName, engine, databaseId)
			err := client.Get(endpoint, res)
			if err != nil {
				if errOvh, ok := err.(*ovh.APIError); ok && errOvh.Code == 404 {
					return res, "DELETED", nil
				} else {
					return res, "", err
				}
			}

			return res, res.Status, nil
		},
		Timeout:    20 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	return err
}
