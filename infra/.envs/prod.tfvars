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

google_analytics_id = "G-CZ6STBPSB2"

frontend_docker_build_target = "static"

backend_domains                      = ["api.webstatus.dev"]
frontend_domains                     = ["webstatus.dev", "www.webstatus.dev"]
custom_ssl_certificates_for_frontend = []
custom_ssl_certificates_for_backend  = []

spanner_region_override = "nam-eur-asia1"

backend_cache_settings = {
  default_duration                  = "1h"
  aggregated_feature_stats_duration = "24h"
}

backend_cors_allowed_origin = "https://*"

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

firebase_api_key_location = "prod-firebase-app-api-key"

auth_github_config_locations = {
  client_id     = "prod-github-client-id"
  client_secret = "prod-github-client-secret"
}

backend_min_instance_count  = 1
frontend_min_instance_count = 1
