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

google_analytics_id = "G-EPZE5TL134"

frontend_docker_build_target = "static"

backend_domains_for_gcp_managed_certificates  = []
frontend_domains_for_gcp_managed_certificates = []

# Temporary for UbP.
ssl_certificates = ["ub-self-sign"]

cache_duration = "5m"

# Needed for UbP.
backend_cors_allowed_origin = "https://website-webstatus-dev.corp.goog"

bcd_region_schedules = {
  "us-central1"  = "0 19 * * *" # Daily at 7:00 PM
  "europe-west1" = "0 7 * * *"  # Daily at 7:00 AM
}

web_features_region_schedules = {
  "us-central1"  = "0 20 * * *" # Daily at 8:00 PM
  "europe-west1" = "0 8 * * *"  # Daily at 8:00 AM
}

chromium_region_schedules = {
  "us-central1"  = "0 21 * * *" # Daily at 9:00 PM
  "europe-west1" = "0 9 * * *"  # Daily at 9:00 AM
}

uma_region_schedules = {
  "us-central1"  = "0 22 * * *" # Daily at 10:00 PM
  "europe-west1" = "0 10 * * *" # Daily at 10:00 AM
}

wpt_region_schedules = {
  "us-central1"  = "0 21 * * *" # Daily at 9:00 PM
  "europe-west1" = "0 9 * * *"  # Daily at 9:00 AM
}
