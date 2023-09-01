variable "env_id" {
  type = string
}

variable "regions" {
  type = list(string)
}

variable "repo_downloader_step_region_to_step_info_map" {
  type = map(object({
    name = string
    url  = string
  }))
}

variable "repo_bucket" {
  type = string
}

variable "firestore_info" {
  type = object({
    database_name = string
    project_id    = string
  })
}

variable "docker_repository_details" {
  type = object({
    hostname = string
    url      = string
  })
}