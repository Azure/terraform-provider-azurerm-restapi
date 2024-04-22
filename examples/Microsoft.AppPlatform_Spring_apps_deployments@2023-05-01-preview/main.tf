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

resource "azapi_resource" "Spring" {
  type      = "Microsoft.AppPlatform/Spring@2023-05-01-preview"
  parent_id = azapi_resource.resourceGroup.id
  name      = var.resource_name
  location  = var.location
  body = {
    properties = {
      zoneRedundant = false
    }
    sku = {
      name = "E0"
    }
  }
  schema_validation_enabled = false
  response_export_values    = ["*"]
}

resource "azapi_resource" "app" {
  type      = "Microsoft.AppPlatform/Spring/apps@2023-05-01-preview"
  parent_id = azapi_resource.Spring.id
  name      = var.resource_name
  location  = var.location
  body = {
    properties = {
      customPersistentDisks = [
      ]
      enableEndToEndTLS = false
      public            = false
    }
  }
  schema_validation_enabled = false
  response_export_values    = ["*"]
}

resource "azapi_resource" "deployment" {
  type      = "Microsoft.AppPlatform/Spring/apps/deployments@2023-05-01-preview"
  parent_id = azapi_resource.app.id
  name      = var.resource_name
  body = {
    properties = {
      deploymentSettings = {
        environmentVariables = {
        }
      }
      source = {
        customContainer = {
          args = [
          ]
          command = [
          ]
          containerImage    = "springio/gs-spring-boot-docker"
          languageFramework = ""
          server            = "docker.io"
        }
        type = "Container"
      }
    }
    sku = {
      capacity = 1
      name     = "E0"
      tier     = "Enterprise"
    }
  }
  schema_validation_enabled = false
  response_export_values    = ["*"]
}

