module "sample_custom_step" {
  source = "./common_workflow_steps/sample_step"

  env_id                    = var.env_id
  regions                   = var.regions
  docker_repository_details = var.docker_repository_details
}

module "repo_downloader_step" {
  source = "./common_workflow_steps/repo_downloader"

  env_id                    = var.env_id
  regions                   = var.regions
  docker_repository_details = var.docker_repository_details
  repo_bucket               = var.buckets.repo_download_bucket
  github_token_secret_id    = var.secret_ids.github_token
}