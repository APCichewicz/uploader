terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id = "017fc95d-05f5-4661-9865-ef0150449c51"
}

resource "random_string" "random_suffix" {
  length = 4
  special = false
  upper = false
}

resource "azurerm_resource_group" "upload_service_rg" {
  name = "upload-service-rg"
  location = "eastus"
}

resource "azurerm_storage_account" "upload_service_storage" {
  name = lower("uploadservice${random_string.random_suffix.result}")
  resource_group_name = azurerm_resource_group.upload_service_rg.name
  location = "eastus"
  account_tier = "Standard"
  account_replication_type = "LRS"  
}

resource "azurerm_storage_container" "upload_service_container" {
  name = "upload-service-container"
  storage_account_id = azurerm_storage_account.upload_service_storage.id
}

output "upload_service_storage_connection_string" {
  value = azurerm_storage_account.upload_service_storage.primary_connection_string
  sensitive = true
}
