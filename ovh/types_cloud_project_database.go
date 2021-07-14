package ovh

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/ovh/terraform-provider-ovh/ovh/helpers"
	"time"
)

type CloudProjectDatabaseCreateOpts struct {
	Description  *string                           `json:"description,omitempty"`
	NodesList    []*CloudProjectDatabaseNode       `json:"nodesList,omitempty"`
	NodesPattern *CloudProjectDatabaseNodesPattern `json:"nodesPattern,omitempty"`
	Plan         string                            `json:"plan"`
	NetworkId    *string                           `json:"networkId,omitempty"`
	SubnetId     *string                           `json:"subnetId,omitempty"`
	Version      string                            `json:"version"`
}

type CloudProjectDatabaseNode struct {
	Flavor string `json:"flavor"`
	Region string `json:"region"`
}

type CloudProjectDatabaseNodesPattern struct {
	Flavor string `json:"flavor"`
	Number int    `json:"number"`
	Region string `json:"region"`
}

func (opts *CloudProjectDatabaseCreateOpts) FromResource(d *schema.ResourceData) *CloudProjectDatabaseCreateOpts {
	opts.Description = helpers.GetNilStringPointerFromData(d, "description")
	opts.Plan = d.Get("plan").(string)

	//nodes := d.Get("nodes_list").([]interface{})
	//if nodes != nil && len(nodes) > 0 {
	//	nodesList := make([]*CloudProjectDatabaseNode, len(nodes))
	//	for i, _ := range nodes {
	//		nodesList[i] = (&CloudProjectDatabaseNode{}).FromResourceWithPath(d, fmt.Sprintf("nodes_list.%d", i))
	//	}
	//	opts.NodesList = nodesList
	//}
	//
	nodesPattern := d.Get("nodes_pattern").(interface{})
	if nodesPattern != nil {
		opts.NodesPattern = (&CloudProjectDatabaseNodesPattern{}).FromResourceWithPath(d, "nodes_pattern.0")
	}

	opts.NetworkId = helpers.GetNilStringPointerFromData(d, "networkId")
	opts.SubnetId = helpers.GetNilStringPointerFromData(d, "subnetId")
	opts.Version = d.Get("version").(string)

	return opts
}

func (opts *CloudProjectDatabaseNode) FromResourceWithPath(d *schema.ResourceData, parent string) *CloudProjectDatabaseNode {
	opts.Flavor = d.Get(fmt.Sprintf("%s.flavor", parent)).(string)
	opts.Region = d.Get(fmt.Sprintf("%s.region", parent)).(string)

	return opts
}

func (opts *CloudProjectDatabaseNodesPattern) FromResourceWithPath(d *schema.ResourceData, parent string) *CloudProjectDatabaseNodesPattern {
	opts.Flavor = d.Get(fmt.Sprintf("%s.flavor", parent)).(string)
	opts.Number = d.Get(fmt.Sprintf("%s.number", parent)).(int)
	opts.Region = d.Get(fmt.Sprintf("%s.region", parent)).(string)

	return opts
}

func (s *CloudProjectDatabaseCreateOpts) String() string {
	return fmt.Sprintf("%s: %s", *s.Description, s.Version)
}

type CloudProjectDatabaseResponse struct {
	CreatedAt   time.Time                `json:"createdAt"`
	Description string                   `json:"description"`
	Domain      string                   `json:"domain"`
	Id          string                   `json:"id"`
	NetworkId   *string                  `json:"networkId,omitempty"`
	NetworkType string                   `json:"networkType"`
	Plan        string                   `json:"plan"`
	PrimaryUser CloudProjectDatabaseUser `json:"primaryUser"`
	NodeNumber  int                      `json:"nodeNumber"`
	Status      string                   `json:"status"`
	SubnetId    *string                  `json:"subnetId,omitempty"`
	Version     string                   `json:"version"`
}

func (v CloudProjectDatabaseResponse) ToMap() map[string]interface{} {
	obj := make(map[string]interface{})
	obj["created_at"] = v.CreatedAt
	obj["description"] = v.Description
	obj["id"] = v.Id

	if v.NetworkId != nil {
		obj["network_id"] = v.NetworkId
	}

	obj["network_type"] = v.NetworkType
	obj["plan"] = v.Plan
	obj["private_network_id"] = v.PrimaryUser
	obj["node_number"] = v.NodeNumber
	obj["status"] = v.Status

	if v.SubnetId != nil {
		obj["subnet_id"] = v.SubnetId
	}

	obj["version"] = v.Version

	return obj
}

type CloudProjectDatabaseUser struct {
	Name     string   `json:"name"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

func (v CloudProjectDatabaseUser) ToMap() map[string]interface{} {
	obj := make(map[string]interface{})
	obj["name"] = v.Name
	obj["password"] = v.Password
	obj["roles"] = v.Roles

	return obj
}

func (s *CloudProjectDatabaseResponse) String() string {
	return fmt.Sprintf("%s(%s): %s", s.Description, s.Id, s.Status)
}
