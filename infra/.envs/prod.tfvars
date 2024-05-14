projects = {
  host     = "web-compass-prod"
  internal = "webstatus-dev-internal-prod"
  public   = "webstatus-dev-public-prod"
}
project_name             = "web-compass-prod"
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
backend_api_url = "https://api.webstatus.dev"
gsi_client_id   = "367048339992-5os99v0p6chosv28dpo9863h9sjeno36.apps.googleusercontent.com"

google_analytics_id = "G-CZ6STBPSB2"

frontend_docker_build_target = "static"

backend_domains_for_gcp_managed_certificates  = ["api.webstatus.dev"]
frontend_domains_for_gcp_managed_certificates = ["webstatus.dev", "www.webstatus.dev"]
ssl_certificates                              = []

spanner_region_override = "nam-eur-asia1"

cache_duration = "5m"

backend_cors_allowed_origin = "https://*"

bcd_region_schedules = {
  "us-central1"  = "0 19 * * *" # Daily at 7:00 PM
  "europe-west1" = "0 7 * * *"  # Daily at 7:00 AM
}

web_features_region_schedules = {
  "us-central1"  = "0 20 * * *" # Daily at 8:00 PM
  "europe-west1" = "0 8 * * *"  # Daily at 8:00 AM
}

wpt_region_schedules = {
  "us-central1"  = "0 21 * * *" # Daily at 9:00 PM
  "europe-west1" = "0 9 * * *"  # Daily at 9:00 AM
}
