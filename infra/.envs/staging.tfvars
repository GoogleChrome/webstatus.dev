projects = {
  host     = "web-compass-staging"
  internal = "webstatus-dev-internal-staging"
  public   = "webstatus-dev-public-staging"
}
project_name             = "web-compass-staging"
spanner_processing_units = 400
deletion_protection      = true
secret_ids = {
  github_token = "jamestestGHtoken-readonly"
}
datastore_region_id = "nam5"
region_information = {
  us-central1 = {
    networks = {
      internal = {
        cidr = "10.1.0.0/16"
      }
      public = {
        cidr = "10.2.0.0/16"
      }
    }
  },
  europe-west1 = {
    networks = {
      internal = {
        cidr = "10.3.0.0/16"
      }
      public = {
        cidr = "10.4.0.0/16"
      }
    }
  },
}
backend_api_url = "https://api-webstatus-dev.corp.goog"
gsi_client_id   = "367048339992-5os99v0p6chosv28dpo9863h9sjeno36.apps.googleusercontent.com"

google_analytics_id = "G-EPZE5TL134"

frontend_docker_build_target = "static"

backend_domains_for_gcp_managed_certificates  = []
frontend_domains_for_gcp_managed_certificates = []

# Temporary for UbP.
ssl_certificates = ["ub-self-sign"]

cache_duration = "5m"

# Needed for UbP.
backend_cors_allowed_origin = "https://website-webstatus-dev.corp.goog"
