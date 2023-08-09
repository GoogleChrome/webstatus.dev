
module "storage" {
  source = "./storage"

  env_id              = var.env_id
  deletion_protection = false
  # `gcloud spanner instance-configs list --project=<PROJECT>` returns the available configs
  spanner_region_id        = coalesce(var.spanner_region_override, "regional-${var.regions[0]}")
  spanner_processing_units = var.spanner_processing_units
  docker_repository_region = coalesce(var.docker_repository_region_override, var.regions[0])
}

module "ingestion" {
  source = "./ingestion"

  env_id                    = var.env_id
  docker_repository_details = module.storage.docker_repository_details
  regions                   = var.regions
}

module "backend" {
  source = "./backend"

  env_id                    = var.env_id
  spanner_datails           = module.storage.spanner_details
  docker_repository_details = module.storage.docker_repository_details
  regions                   = var.regions
  firestore_datails         = module.storage.firestore_details
}

module "frontend" {
  source = "./frontend"

  env_id                    = var.env_id
  docker_repository_details = module.storage.docker_repository_details
  regions                   = var.regions
  backend_api_host          = "TODO"
}