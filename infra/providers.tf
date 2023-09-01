terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.81.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.0.2"
    }
  }
}

provider "google" {
  project = var.project_name
  region  = "us-central1"
}


provider "docker" {
  host = "unix:///var/run/docker.sock"
  registry_auth {
    address     = module.storage.docker_repository_details.hostname
    config_file = pathexpand("~/.docker/config.json")
  }
}