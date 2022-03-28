package portal_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
)

type PortalDashboardDataSource struct{}

func TestAccDataSourcePortalDashboard_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_portal_dashboard", "test")
	r := PortalDashboardDataSource{}
	data.DataSourceTest(t, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).Key("name").HasValue("my-test-dashboard"),
				check.That(data.ResourceName).Key("location").HasValue(data.Locations.Primary),
			),
		},
	})
}

func TestAccDataSourcePortalDashboard_complete(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_portal_dashboard", "test")
	r := PortalDashboardDataSource{}
	data.DataSourceTest(t, []acceptance.TestStep{
		{
			Config: r.complete(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).Key("name").HasValue("my-test-dashboard"),
				check.That(data.ResourceName).Key("location").HasValue(data.Locations.Primary),
				check.That(data.ResourceName).Key("tags.%").HasValue("1"),
				check.That(data.ResourceName).Key("tags.ENV").HasValue("Test"),
				check.That(data.ResourceName).Key("dashboard_properties").HasValue("{\"lenses\":{\"0\":{\"order\":0,\"parts\":{\"0\":{\"metadata\":{\"inputs\":[],\"settings\":{\"content\":{\"settings\":{\"content\":\"## This is only a test :)\",\"subtitle\":\"\",\"title\":\"Test MD Tile\"}}},\"type\":\"Extension/HubsExtension/PartType/MarkdownPart\"},\"position\":{\"colSpan\":3,\"rowSpan\":2,\"x\":0,\"y\":0}}}}}}"),
			),
		},
	})
}

func (PortalDashboardDataSource) basic(data acceptance.TestData) string {
	resourceName := "azurerm_portal_dashboard"
	if !features.ThreePointOhBeta() {
		resourceName = "azurerm_dashboard"
	}
	return fmt.Sprintf(`

%s

data "azurerm_portal_dashboard" "test" {
  name                = %s.test.name
  resource_group_name = azurerm_resource_group.test.name
}
`, PortalDashboardResource{}.basic(data), resourceName)
}

func (PortalDashboardDataSource) complete(data acceptance.TestData) string {
	resourceName := "azurerm_portal_dashboard"
	if !features.ThreePointOhBeta() {
		resourceName = "azurerm_dashboard"
	}
	return fmt.Sprintf(`

%s

data "azurerm_portal_dashboard" "test" {
  name                = %s.test.name
  resource_group_name = azurerm_resource_group.test.name
}
`, PortalDashboardResource{}.complete(data), resourceName)
}
