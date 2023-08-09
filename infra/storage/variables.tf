variable "env_id" {
  type = string
}

variable "deploy_spanner" {
  type    = bool
  default = true
}


variable "spanner_region_id" {
  type        = string
  nullable    = true
  description = "Configuration from https://cloud.google.com/spanner/docs/instance-configurations#available-configurations-multi-region"
}

variable "spanner_processing_units" {
  type = number
}

variable "deletion_protection" {
  type = bool
}

variable "docker_repository_region" {
  type = string
}