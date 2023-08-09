module "sample_custom_step" {
  source = "./common_custom_workflow_steps/sample_step"

  env_id                    = var.env_id
  regions                   = var.regions
  docker_repository_details = var.docker_repository_details
}