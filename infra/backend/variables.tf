variable "spanner_datails" {
  type = object({
    instance = string
    database = string
  })
}

variable "firestore_info" {
  type = object({
    database_name = string
    project_id    = string
  })
}


variable "env_id" {
  type = string
}

variable "regions" {
  type = list(string)
}

variable "docker_repository_details" {
  type = object({
    hostname = string
    url      = string
  })
}