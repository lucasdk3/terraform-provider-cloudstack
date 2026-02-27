package cloudstack

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccKubernetesClusterConfigDataSourceConfig_basic = `
resource "cloudstack_kubernetes_cluster" "test" {
  name                = "tf-k8s-test"
  kubernetes_version  = "1.27.2"
  service_offering_id = "small"
  zone                = "Sandbox-simulator"
  network_id          = "default"
}

data "cloudstack_kubernetes_cluster_config" "test" {
  id = cloudstack_kubernetes_cluster.test.id

  depends_on = [
    cloudstack_kubernetes_cluster.test
  ]
}
`

func TestAccKubernetesClusterConfigDataSource_basic(t *testing.T) {
	resourceName := "cloudstack_kubernetes_cluster.test"
	datasourceName := "data.cloudstack_kubernetes_cluster_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterConfigDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(

					// ID do datasource deve ser igual ao resource
					resource.TestCheckResourceAttrPair(
						datasourceName, "id",
						resourceName, "id",
					),

					// Name deve ser igual
					resource.TestCheckResourceAttrPair(
						datasourceName, "name",
						resourceName, "name",
					),

					// config_data deve existir
					resource.TestCheckResourceAttrSet(
						datasourceName, "config_data",
					),
				),
			},
		},
	})
}
