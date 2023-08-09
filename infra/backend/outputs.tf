locals {
  backend_regional_info = tolist([
    for service in google_cloud_run_v2_service.service :
    {
      "url" : service.uri
      "name" : service.name
    }
  ])
}

output "region_to_backend_info_map" {
  value = zipmap(var.regions, local.backend_regional_info)
}

# output "backend_dns_host" {
#   value = 
#   description = "High level DNS Host"
# }