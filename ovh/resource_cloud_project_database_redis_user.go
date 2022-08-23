package ovh

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/ovh/terraform-provider-ovh/ovh/helpers"
)

func resourceCloudProjectDatabasepRedisUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudProjectDatabaseRedisUserCreate,
		Read:   resourceCloudProjectDatabaseRedisUserRead,
		Delete: resourceCloudProjectDatabaseRedisUserDelete,
		Update: resourceCloudProjectDatabaseRedisUserUpdate,

		Importer: &schema.ResourceImporter{
			State: resourceCloudProjectDatabaseRedisUserImportState,
		},

		Schema: map[string]*schema.Schema{
			"service_name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_CLOUD_PROJECT_SERVICE", nil),
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Description: "Id of the database cluster",
				ForceNew:    true,
				Required:    true,
			},
			"categories": {
				Type:        schema.TypeList,
				Description: "Categories of the user",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"commands": {
				Type:        schema.TypeList,
				Description: "Commands of the user",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"keys": {
				Type:        schema.TypeList,
				Description: "Keys of the user",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the user",
				ForceNew:    true,
				Required:    true,
			},

			//Optional/Computed
			"channels": {
				Type:        schema.TypeList,
				Description: "Channels of the user",
				Optional:    true,
				// If no channels list, channels = ["*"] is computed at creation
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			//Computed
			"created_at": {
				Type:        schema.TypeString,
				Description: "Date of the creation of the user",
				Computed:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "Password of the user",
				Sensitive:   true,
				Computed:    true,
			},
			"status": {
				Type:        schema.TypeString,
				Description: "Current status of the user",
				Computed:    true,
			},
		},
	}
}

func resourceCloudProjectDatabaseRedisUserImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	givenId := d.Id()
	n := 3
	splitId := strings.SplitN(givenId, "/", n)
	if len(splitId) != n {
		return nil, fmt.Errorf("Import Id is not service_name/cluster_id/id formatted")
	}
	serviceName := splitId[0]
	clusterId := splitId[1]
	id := splitId[2]
	d.SetId(id)
	d.Set("cluster_id", clusterId)
	d.Set("service_name", serviceName)

	results := make([]*schema.ResourceData, 1)
	results[0] = d
	return results, nil
}

func resourceCloudProjectDatabaseRedisUserCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	clusterId := d.Get("cluster_id").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/database/redis/%s/user",
		url.PathEscape(serviceName),
		url.PathEscape(clusterId),
	)
	params := (&CloudProjectDatabaseRedisUserCreateOpts{}).FromResource(d)
	res := &CloudProjectDatabaseRedisUserResponse{}

	log.Printf("[DEBUG] Will create user: %+v for cluster %s from project %s", params, clusterId, serviceName)
	err := config.OVHClient.Post(endpoint, params, res)
	if err != nil {
		return fmt.Errorf("calling Post %s with params %+v:\n\t %q", endpoint, params, err)
	}

	log.Printf("[DEBUG] Waiting for user %s to be READY", res.Id)
	err = waitForCloudProjectDatabaseUserReady(config.OVHClient, serviceName, "redis", clusterId, res.Id, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return fmt.Errorf("timeout while waiting user %s to be READY: %w", res.Id, err)
	}
	log.Printf("[DEBUG] user %s is READY", res.Id)

	d.SetId(res.Id)
	d.Set("password", res.Password)

	return resourceCloudProjectDatabaseRedisUserRead(d, meta)
}

func resourceCloudProjectDatabaseRedisUserRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	clusterId := d.Get("cluster_id").(string)
	id := d.Id()

	endpoint := fmt.Sprintf("/cloud/project/%s/database/redis/%s/user/%s",
		url.PathEscape(serviceName),
		url.PathEscape(clusterId),
		url.PathEscape(id),
	)
	res := &CloudProjectDatabaseRedisUserResponse{}

	log.Printf("[DEBUG] Will read user %s from cluster %s from project %s", id, clusterId, serviceName)
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

	log.Printf("[DEBUG] Read user %+v", res)
	return nil
}

func resourceCloudProjectDatabaseRedisUserUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	clusterId := d.Get("cluster_id").(string)
	id := d.Id()

	endpoint := fmt.Sprintf("/cloud/project/%s/database/redis/%s/user/%s",
		url.PathEscape(serviceName),
		url.PathEscape(clusterId),
		url.PathEscape(id),
	)
	params := (&CloudProjectDatabaseRedisUserUpdateOpts{}).FromResource(d)

	log.Printf("[DEBUG] Will update user: %+v from cluster %s from project %s", params, clusterId, serviceName)
	err := config.OVHClient.Put(endpoint, params, nil)
	if err != nil {
		return fmt.Errorf("calling Put %s with params %+v:\n\t %q", endpoint, params, err)
	}

	log.Printf("[DEBUG] Waiting for user %s to be READY", id)
	err = waitForCloudProjectDatabaseUserReady(config.OVHClient, serviceName, "redis", clusterId, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return fmt.Errorf("timeout while waiting user %s to be READY: %w", id, err)
	}
	log.Printf("[DEBUG] user %s is READY", id)

	return resourceCloudProjectDatabaseRedisUserRead(d, meta)
}

func resourceCloudProjectDatabaseRedisUserDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)
	clusterId := d.Get("cluster_id").(string)
	id := d.Id()

	endpoint := fmt.Sprintf("/cloud/project/%s/database/redis/%s/user/%s",
		url.PathEscape(serviceName),
		url.PathEscape(clusterId),
		url.PathEscape(id),
	)

	log.Printf("[DEBUG] Will delete useruser %s from cluster %s from project %s", id, clusterId, serviceName)
	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		return helpers.CheckDeleted(d, err, endpoint)
	}

	log.Printf("[DEBUG] Waiting for user %s to be DELETED", id)
	err = waitForCloudProjectDatabaseUserDeleted(config.OVHClient, serviceName, "redis", clusterId, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return fmt.Errorf("timeout while waiting user %s to be DELETED: %w", id, err)
	}
	log.Printf("[DEBUG] user %s is DELETED", id)

	d.SetId("")

	return nil
}
