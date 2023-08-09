locals {
  step_regional_info = tolist([
    for service in google_cloud_run_v2_service.service :
    {
      "url" : service.uri
      "name" : service.name
    }
  ])
}

output "region_to_step_info_map" {
  value = zipmap(var.regions, local.step_regional_info)
}