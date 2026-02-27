package cloudstack

import (
	"fmt"
	"log"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudstackKubernetesClusterConfig() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudstackKubernetesClusterConfigRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"config_data": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true, // IMPORTANTE pois é kubeconfig
			},
		},
	}
}

func dataSourceCloudstackKubernetesClusterConfigRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	clusterID := d.Get("id").(string)

	p := cs.Kubernetes.NewGetKubernetesClusterConfigParams()
	p.SetId(clusterID)

	resp, err := cs.Kubernetes.GetKubernetesClusterConfig(p)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes cluster config: %s", err)
	}

	if resp == nil {
		return fmt.Errorf("cluster config not found for ID %s", clusterID)
	}

	log.Printf("[DEBUG] Found Kubernetes cluster config for cluster: %s", resp.Name)

	d.SetId(resp.Id)
	d.Set("name", resp.Name)
	d.Set("config_data", resp.Configdata)

	return nil
}
