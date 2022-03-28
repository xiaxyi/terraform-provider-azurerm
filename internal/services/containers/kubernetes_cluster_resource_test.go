package containers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/containers/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

type KubernetesClusterResource struct{}

var (
	olderKubernetesVersion   = "1.21.7"
	currentKubernetesVersion = "1.22.4"
)

func TestAccKubernetesCluster_hostEncryption(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_kubernetes_cluster", "test")
	r := KubernetesClusterResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.hostEncryption(data, currentKubernetesVersion),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("default_node_pool.0.enable_host_encryption").HasValue("true"),
			),
		},
	})
}

func (t KubernetesClusterResource) Exists(ctx context.Context, clients *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := parse.ClusterID(state.ID)
	if err != nil {
		return nil, err
	}

	resp, err := clients.Containers.KubernetesClustersClient.Get(ctx, id.ResourceGroup, id.ManagedClusterName)
	if err != nil {
		return nil, fmt.Errorf("reading Kubernetes Cluster (%s): %+v", id.String(), err)
	}

	return utils.Bool(resp.ID != nil), nil
}

func (KubernetesClusterResource) updateDefaultNodePoolAgentCount(nodeCount int) acceptance.ClientCheckFunc {
	return func(ctx context.Context, clients *clients.Client, state *pluginsdk.InstanceState) error {
		nodePoolName := state.Attributes["default_node_pool.0.name"]
		clusterName := state.Attributes["name"]
		resourceGroup := state.Attributes["resource_group_name"]

		nodePool, err := clients.Containers.AgentPoolsClient.Get(ctx, resourceGroup, clusterName, nodePoolName)
		if err != nil {
			return fmt.Errorf("Bad: Get on agentPoolsClient: %+v", err)
		}

		if nodePool.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Node Pool %q (Kubernetes Cluster %q / Resource Group: %q) does not exist", nodePoolName, clusterName, resourceGroup)
		}

		if nodePool.ManagedClusterAgentPoolProfileProperties == nil {
			return fmt.Errorf("Bad: Node Pool %q (Kubernetes Cluster %q / Resource Group: %q): `properties` was nil", nodePoolName, clusterName, resourceGroup)
		}

		nodePool.ManagedClusterAgentPoolProfileProperties.Count = utils.Int32(int32(nodeCount))

		future, err := clients.Containers.AgentPoolsClient.CreateOrUpdate(ctx, resourceGroup, clusterName, nodePoolName, nodePool)
		if err != nil {
			return fmt.Errorf("Bad: updating node pool %q: %+v", nodePoolName, err)
		}

		if err := future.WaitForCompletionRef(ctx, clients.Containers.AgentPoolsClient.Client); err != nil {
			return fmt.Errorf("Bad: waiting for update of node pool %q: %+v", nodePoolName, err)
		}

		return nil
	}
}

func (KubernetesClusterResource) hostEncryption(data acceptance.TestData, controlPlaneVersion string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-aks-%d"
  location = "%s"
}

resource "azurerm_kubernetes_cluster" "test" {
  name                = "acctestaks%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  dns_prefix          = "acctestaks%d"
  kubernetes_version  = %q

  default_node_pool {
    name                   = "default"
    node_count             = 1
    vm_size                = "Standard_DS2_v2"
    enable_host_encryption = true
  }

  identity {
    type = "SystemAssigned"
  }
}
  `, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger, controlPlaneVersion)
}

func (r KubernetesClusterResource) upgradeSettingsConfig(data acceptance.TestData, maxSurge string) string {
	if maxSurge != "" {
		maxSurge = fmt.Sprintf(`upgrade_settings {
    max_surge = %q
  }`, maxSurge)
	}

	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-aks-%d"
  location = "%s"
}

resource "azurerm_kubernetes_cluster" "test" {
  name                = "acctestaks%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  dns_prefix          = "acctestaks%d"

  default_node_pool {
    name       = "default"
    node_count = 1
    vm_size    = "Standard_DS2_v2"
    %s
  }

  identity {
    type = "SystemAssigned"
  }
}
  `, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger, maxSurge)
}
