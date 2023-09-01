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

variable "repo_bucket" {
  type = string
}

variable "github_token_secret_id" {
  type = string
}