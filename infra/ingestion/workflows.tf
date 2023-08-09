module "sample_workflow" {
  source = "./workflows/sample"

  regions                                    = var.regions
  env_id                                     = var.env_id
  sample_custom_step_region_to_step_info_map = module.sample_custom_step.region_to_step_info_map
}