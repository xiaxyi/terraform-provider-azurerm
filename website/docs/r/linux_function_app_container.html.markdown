---
subcategory: "App Service (Web Apps)"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_linux_function_app_on_container"
description: |-
Manages a Containerized Linux Function App on Azure Container Apps.
---

# azurerm_linux_function_app_on_container

Manages a Containerized Linux Function App on Azure Container Apps.

## Example Usage

```hcl
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_storage_account" "example" {
  name                     = "linuxfunctionappsa"
  resource_group_name      = azurerm_resource_group.example.name
  location                 = azurerm_resource_group.example.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_container_app_environment" "example" {
  name                       = "Example-Environment"
  location                   = azurerm_resource_group.example.location
  resource_group_name        = azurerm_resource_group.example.name
}

resource "azurerm_linux_function_app_on_container" "example" {
  name                = "example-linux-function-app-on-container"
  resource_group_name = azurerm_resource_group.example.name
  container_app_environment_id = azurerm_container_app_environment.example.id

  storage_account_name       = azurerm_storage_account.example.name
  storage_account_access_key = azurerm_storage_account.example.primary_access_key

 registry {
    registry_server_url = "docker.io"
registry_username   = "username"
    registry_password   = "pwd"
  }
    container_image     = "docker/getting-started:latest"

  site_config {
ftps_state        = "Disabled"
use_32_bit_worker = false
}
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The name which should be used for this Containerized Linux Function App. Changing this forces a new Containerized Linux Function App to be created. Limit the function name to 32 characters to avoid naming collisions. For more information about [Function App naming rule](https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules#microsoftweb) and [Host ID Collisions](https://github.com/Azure/azure-functions-host/wiki/Host-IDs#host-id-collisions)

* `resource_group_name` - (Required) The name of the Resource Group where the Containerized Linux Function App should exist. Changing this forces a new Containerized Linux Function App to be created.

* `container_app_environment_id` - (Required) The ID of the Container App Environment within which this Containerized Linux Function App should exist. Changing this forces a new resource to be created.

* `site_config` - (Required) A `site_config` block as defined below.

---

* `app_settings` - (Optional) A map of key-value pairs for [App Settings](https://docs.microsoft.com/azure/azure-functions/functions-app-settings) and custom values.

~> **Note:** Please use `functions_extension_version` to set the function runtime version, terraform will assign the values to the key `FUNCTIONS_EXTENSION_VERSION` in app setting.

~> **Note:** For storage related settings, please use related properties that are available such as `storage_account_access_key`, terraform will assign the value to keys such as `WEBSITE_CONTENTAZUREFILECONNECTIONSTRING`, `AzureWebJobsStorage` in app_setting.

~> **Note:** For application insight related settings, please use `application_insights_connection_string` and `application_insights_key`, terraform will assign the value to the key `APPINSIGHTS_INSTRUMENTATIONKEY` and `APPLICATIONINSIGHTS_CONNECTION_STRING` in app setting.

* `builtin_logging_enabled` - (Optional) Should built in logging be enabled. Configures `AzureWebJobsDashboard` app setting based on the configured storage setting. Defaults to `true`.

* `functions_extension_version` - (Optional) The runtime version associated with the Function App. Defaults to `~4`.

* `key_vault_reference_identity_id` - (Optional) The User Assigned Identity ID used for accessing KeyVault secrets. The identity must be assigned to the application in the `identity` block. [For more information see - Access vaults with a user-assigned identity](https://docs.microsoft.com/azure/app-service/app-service-key-vault-references#access-vaults-with-a-user-assigned-identity)

* `storage_account` - (Optional) One or more `storage_account` blocks as defined below.

* `storage_account_access_key` - (Optional) The access key which will be used to access the backend storage account for the Function App. Conflicts with `storage_uses_managed_identity`.

* `storage_account_name` - (Optional) The backend storage account name which will be used by this Function App.

* `storage_key_vault_secret_id` - (Optional) The Key Vault Secret ID, optionally including version, that contains the Connection String to connect to the storage account for this Function App.

~> **NOTE:** `storage_key_vault_secret_id` cannot be used with `storage_account_name`.

~> **NOTE:** `storage_key_vault_secret_id` used without a version will use the latest version of the secret, however, the service can take up to 24h to pick up a rotation of the latest version. See the [official docs](https://docs.microsoft.com/azure/app-service/app-service-key-vault-references#rotation) for more information.

* `tags` - (Optional) A mapping of tags which should be assigned to the Containerized Linux Function App.

---

A `registry` block supports the following:

* `registry_url` - (Required) The URL of the docker registry.

* `image_name` - (Required) The name of the Docker image to use.

* `image_tag` - (Required) The image tag of the image to use.

* `registry_username` - (Optional) The username to use for connections to the registry.

* `registry_password` - (Optional) The password for the account to use to connect to the registry.

---

A `site_config` block supports the following:

* `application_insights_connection_string` - (Optional) The Connection String for linking the Containerized Linux Function App to Application Insights.

* `application_insights_key` - (Optional) The Instrumentation Key for connecting the Containerized Linux Function App to Application Insights.

* `ftps_state` - (Optional) State of FTP / FTPS service for this function app. Possible values include: `AllAllowed`, `FtpsOnly` and `Disabled`. Defaults to `Disabled`.

* `minimum_tls_version` - (Optional) The configures the minimum version of TLS required for SSL requests. Possible values include: `1.0`, `1.1`, and `1.2`. Defaults to `1.2`.

* `use_32_bit_worker` - (Optional) Should the Linux Web App use a 32-bit worker process. Defaults to `false`.

---

A `storage_account` block supports the following:

* `access_key` - (Required) The Access key for the storage account.

* `account_name` - (Required) The Name of the Storage Account.

* `name` - (Required) The name which should be used for this Storage Account.

* `share_name` - (Required) The Name of the File Share or Container Name for Blob storage.

* `type` - (Required) The Azure Storage Type. Possible values include `AzureFiles` and `AzureBlob`.

* `mount_path` - (Optional) The path at which to mount the storage share.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The ID of the Containerized Linux Function App.

* `custom_domain_verification_id` - The identifier used by App Service to perform domain ownership verification via DNS TXT record.

* `default_hostname` - The default hostname of the Containerized Linux Function App.

* `hosting_environment_id` - The ID of the App Service Environment used by Function App.

* `identity` - An `identity` block as defined below.

* `kind` - The Kind value for this Containerized Linux Function App.

* `outbound_ip_address_list` - A list of outbound IP addresses. For example `["52.23.25.3", "52.143.43.12"]`

* `outbound_ip_addresses` - A comma separated list of outbound IP addresses as a string. For example `52.23.25.3,52.143.43.12`.

* `possible_outbound_ip_address_list` - A list of possible outbound IP addresses, not all of which are necessarily in use. This is a superset of `outbound_ip_address_list`. For example `["52.23.25.3", "52.143.43.12"]`.

* `possible_outbound_ip_addresses` - A comma separated list of possible outbound IP addresses as a string. For example `52.23.25.3,52.143.43.12,52.143.43.17`. This is a superset of `outbound_ip_addresses`.

* `site_credential` - A `site_credential` block as defined below.

---

An `identity` block exports the following:

* `principal_id` - The Principal ID associated with this Managed Service Identity.

* `tenant_id` - The Tenant ID associated with this Managed Service Identity.

---

A `site_credential` block exports the following:

* `name` - The Site Credentials Username used for publishing.

* `password` - The Site Credentials Password used for publishing.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Containerized Linux Function App.
* `read` - (Defaults to 5 minutes) Used when retrieving the Containerized Linux Function App.
* `update` - (Defaults to 30 minutes) Used when updating the Containerized Linux Function App.
* `delete` - (Defaults to 30 minutes) Used when deleting the Containerized Linux Function App.

## Import

Containerized Linux Function Apps can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_linux_function_app.example /subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resGroup1/providers/Microsoft.Web/sites/site1
```
