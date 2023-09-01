module "sample_workflow" {
  source = "./workflows/sample"

  regions                                    = var.regions
  env_id                                     = var.env_id
  sample_custom_step_region_to_step_info_map = module.sample_custom_step.region_to_step_info_map
}

module "web_features_repo_workflow" {
  source = "./workflows/web_features_repo"

  regions                                      = var.regions
  env_id                                       = var.env_id
  repo_downloader_step_region_to_step_info_map = module.repo_downloader_step.region_to_step_info_map
  firestore_info                               = var.firestore_info
  repo_bucket                                  = var.buckets.repo_download_bucket
  docker_repository_details                    = var.docker_repository_details
}