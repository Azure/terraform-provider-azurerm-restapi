terraform {
  required_providers {
    azapi = {
      source = "Azure/azapi"
    }
  }
}

provider "azapi" {
  skip_provider_registration = false
}

variable "resource_name" {
  type    = string
  default = "acctest0001"
}

variable "location" {
  type    = string
  default = "westeurope"
}

resource "azapi_resource" "resourceGroup" {
  type                      = "Microsoft.Resources/resourceGroups@2020-06-01"
  name                      = var.resource_name
  location                  = var.location
}

resource "azapi_resource" "managedCluster" {
  type      = "Microsoft.ContainerService/managedClusters@2023-04-02-preview"
  parent_id = azapi_resource.resourceGroup.id
  name      = var.resource_name
  location  = var.location
  body = jsonencode({
    identity = {
      type                   = "SystemAssigned"
      userAssignedIdentities = null
    }
    properties = {
      agentPoolProfiles = [
        {
          count  = 1
          mode   = "System"
          name   = "default"
          vmSize = "Standard_DS2_v2"
        },
      ]
      dnsPrefix = var.resource_name
    }
  })
  schema_validation_enabled = false
  response_export_values    = ["*"]
  ignore_changes            = ["properties.agentPoolProfiles"]
}

resource "azapi_resource" "agentPool" {
  type      = "Microsoft.ContainerService/managedClusters/agentPools@2023-04-02-preview"
  parent_id = azapi_resource.managedCluster.id
  name      = "internal"
  body = jsonencode({
    properties = {
      count  = 1
      mode   = "User"
      vmSize = "Standard_DS2_v2"
    }
  })
  schema_validation_enabled = false
  response_export_values    = ["*"]
}

