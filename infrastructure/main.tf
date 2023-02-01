terraform {
  required_version = ">= 1.3.7"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 4.51.0"
    }
  }
  backend "gcs" {
    bucket = "gsr-tf-state"
    prefix = "tf-state"
  }
}

provider "google" {
  project = "go-semantic-release"
  region  = "europe-west1"
}

locals {
  stages = toset(["staging"])
}


module "gcr" {
  for_each = local.stages
  source   = "./gcr"
  name     = each.value
}
