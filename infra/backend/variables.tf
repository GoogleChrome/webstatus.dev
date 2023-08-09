variable "spanner_datails" {
  type = object({
    instance = string
    database = string
  })
}

variable "firestore_datails" {
  type = object({
    database = string
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