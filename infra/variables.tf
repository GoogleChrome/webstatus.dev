variable "project_name" {
  type        = string
  description = "The ID of the Google Cloud project"
}

variable "primary_region" {
  type        = string
  description = "Primary region. Useful for ."
}

variable "regions" {
  type = list(string)
}

variable "spanner_region_override" {
  type     = string
  nullable = true
  default  = null
}

variable "spanner_processing_units" {
  type = number
}

variable "deletion_protection" {
  type        = bool
  description = "Protect applicable resources from deletion."
}

variable "env_id" {
  description = "Environment ID. Commonly dervied from the branch name"
  type        = string
}

variable "docker_repository_region_override" {
  type     = string
  nullable = true
  default  = null
}

locals {
  docker_repository_region = coalesce(var.docker_repository_region_override, var.regions[0])
}

variable "secret_ids" {
  type = object({
    github_token = string
  })
}