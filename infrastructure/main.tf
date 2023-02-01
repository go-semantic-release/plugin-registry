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
  defaultEnvSecrets = {
    CLOUDFLARE_R2_ACCESS_KEY_ID     = "cloudflare-access-key-id"
    CLOUDFLARE_R2_SECRET_ACCESS_KEY = "cloudflare-secret-access-key"
    CLOUDFLARE_ACCOUNT_ID           = "cloudflare-account-id"
    ADMIN_ACCESS_TOKEN              = "admin-access-token"
  }
}

locals {
  stages = toset(["staging", "production"])
  envSecrets = {
    staging = merge({
      PLUGIN_CACHE_HOST    = "staging-plugin-cache-host"
      GITHUB_TOKEN         = "staging-github"
      CLOUDFLARE_R2_BUCKET = "staging-cloudflare-r2-bucket"
    }, local.defaultEnvSecrets)
    production = merge({
      PLUGIN_CACHE_HOST    = "production-plugin-cache-host"
      GITHUB_TOKEN         = "production-github"
      CLOUDFLARE_R2_BUCKET = "production-cloudflare-r2-bucket"
    }, local.defaultEnvSecrets)
  }
  domains = {
    staging    = "registry-staging.go-semantic-release.xyz"
    production = "registry.go-semantic-release.xyz"
  }
}

module "gcr" {
  for_each   = local.stages
  source     = "./gcr"
  stage      = each.value
  envSecrets = local.envSecrets[each.value]
  domain     = local.domains[each.value]
}
