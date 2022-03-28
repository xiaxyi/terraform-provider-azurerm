---
subcategory: "Policy"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_resource_policy_remediation"
description: |-
  Manages an Azure Resource Policy Remediation.
---

# azurerm_resource_policy_remediation

Manages an Azure Resource Policy Remediation.

## Example Usage

```hcl
resource "azurerm_resource_group" "example" {
  name     = "resourcegroup1"
  location = "West US"
}

resource "azurerm_virtual_network" "example" {
  name                = "vnet1"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  address_space       = ["10.0.0.0/16"]
}

resource "azurerm_resource_policy_assignment" "example" {
  name                 = "assignment1"
  resource_id          = azurerm_virtual_network.example.id
  policy_definition_id = data.azurerm_policy_definition.example.id
  parameters = jsonencode({
    "listOfAllowedLocations" = {
      "value" = [azurerm_resource_group.example.location, "East US"]
    }
  })
}

resource "azurerm_resource_policy_remediation" "example" {
  name                 = "remediation1"
  resource_id          = azurerm_virtual_network.example.id
  policy_assignment_id = azurerm_resource_group_policy_assignment.test.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Policy Remediation. Changing this forces a new resource to be created.

* `resource_id` - (Required) The Resource ID at which the Policy Remediation should be applied. Changing this forces a new resource to be created.

* `policy_assignment_id` - (Required) The ID of the Policy Assignment that should be remediated.

* `policy_definition_reference_id` - (Optional) The unique ID for the policy definition within the policy set definition that should be remediated. Required when the policy assignment being remediated assigns a policy set definition.

* `location_filters` - (Optional) A list of the resource locations that will be remediated.

* `resource_discovery_mode` - (Optional) The way that resources to remediate are discovered. Possible values are `ExistingNonCompliant`, `ReEvaluateCompliance`. Defaults to `ExistingNonCompliant`.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Policy Remediation.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Policy Remediation.
* `update` - (Defaults to 30 minutes) Used when updating the Policy Remediation.
* `read` - (Defaults to 5 minutes) Used when retrieving the Policy Remediation.
* `delete` - (Defaults to 30 minutes) Used when deleting the Policy Remediation.


## Import

Policy Remediations can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_policy_remediation.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.PolicyInsights/remediations/remediation1
```
