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
variable "buckets" {
  type = object({
    repo_download_bucket = string
  })
}

variable "secret_ids" {
  type = object({
    github_token = string
  })
}

variable "firestore_info" {
  type = object({
    database_name = string
    project_id    = string
  })
}