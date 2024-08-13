---
page_title: "azapi_resource Data Source - terraform-provider-azapi"
subcategory: ""
description: |-
  This resource can access any existing Azure resource manager resource.
---

# azapi_resource (Data Source)

This resource can access any existing Azure resource manager resource.## Example Usage

```terraform
terraform {
  required_providers {
    azapi = {
      source = "Azure/azapi"
    }
  }
}

provider "azapi" {
  enable_hcl_output_for_data_source = true
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "example-rg"
  location = "west europe"
}

resource "azurerm_container_registry" "example" {
  name                = "example"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  sku                 = "Premium"
  admin_enabled       = false
}

data "azapi_resource" "example" {
  name      = "example"
  parent_id = azurerm_resource_group.example.id
  type      = "Microsoft.ContainerRegistry/registries@2020-11-01-preview"

  response_export_values = ["properties.loginServer", "properties.policies.quarantinePolicy.status"]
}

// it will output "registry1.azurecr.io"
output "login_server" {
  value = data.azapi_resource.example.output.properties.loginServer
}

// it will output "disabled"
output "quarantine_policy" {
  value = data.azapi_resource.example.output.properties.policies.quarantinePolicy.status
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `type` (String) In a format like `<resource-type>@<api-version>`. `<resource-type>` is the Azure resource type, for example, `Microsoft.Storage/storageAccounts`. `<api-version>` is version of the API used to manage this azure resource.

### Optional

- `name` (String) Specifies the name of the Azure resource.
- `parent_id` (String) The ID of the azure resource in which this resource is created. It supports different kinds of deployment scope for **top level** resources:

  - resource group scope: `parent_id` should be the ID of a resource group, it's recommended to manage a resource group by azurerm_resource_group.
	- management group scope: `parent_id` should be the ID of a management group, it's recommended to manage a management group by azurerm_management_group.
	- extension scope: `parent_id` should be the ID of the resource you're adding the extension to.
	- subscription scope: `parent_id` should be like \x60/subscriptions/00000000-0000-0000-0000-000000000000\x60
	- tenant scope: `parent_id` should be /

  For child level resources, the `parent_id` should be the ID of its parent resource, for example, subnet resource's `parent_id` is the ID of the vnet.

  For type `Microsoft.Resources/resourceGroups`, the `parent_id` could be omitted, it defaults to subscription ID specified in provider or the default subscription (You could check the default subscription by azure cli command: `az account show`).
- `resource_id` (String) The ID of the Azure resource to retrieve.
- `response_export_values` (List of String) A list of path that needs to be exported from response body. Setting it to `["*"]` will export the full response body. Here's an example. If it sets to `["properties.loginServer", "properties.policies.quarantinePolicy.status"]`, it will set the following HCL object to computed property output.

	```text
	{
		properties = {
			loginServer = "registry1.azurecr.io"
			policies = {
				quarantinePolicy = {
					status = "disabled"
				}
			}
		}
	}
	```
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) The ID of the Azure resource.
- `identity` (Attributes List) (see [below for nested schema](#nestedatt--identity))
- `location` (String) The location of the Azure resource.
- `output` (Dynamic) The output HCL object containing the properties specified in `response_export_values`. Here are some examples to use the values.

	```terraform
	// it will output "registry1.azurecr.io"
	output "login_server" {
		value = data.azapi_resource.example.output.properties.loginServer
	}

	// it will output "disabled"
	output "quarantine_policy" {
		value = data.azapi_resource.example.output.properties.policies.quarantinePolicy.status
	}
	```
- `tags` (Map of String) A mapping of tags which are assigned to the Azure resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Read operations occur during any refresh or planning operation when refresh is enabled.


<a id="nestedatt--identity"></a>
### Nested Schema for `identity`

Read-Only:

- `identity_ids` (List of String) A list of User Managed Identity ID's which should be assigned to the azure resource.
- `principal_id` (String) The Principal ID for the Service Principal associated with the Managed Service Identity of this Azure resource.
- `tenant_id` (String) The Tenant ID for the Service Principal associated with the Managed Service Identity of this Azure resource.
- `type` (String) The Type of Identity which should be used for this azure resource. Possible values are `SystemAssigned`, `UserAssigned` and `SystemAssigned,UserAssigned`