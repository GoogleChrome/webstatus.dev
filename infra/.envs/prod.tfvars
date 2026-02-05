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
frontend_base_url                    = "https://webstatus.dev"
custom_ssl_certificates_for_frontend = []
custom_ssl_certificates_for_backend  = []

spanner_region_override = "nam-eur-asia1"

backend_cache_settings = {
  default_duration                  = "1h"
  aggregated_feature_stats_duration = "24h"
}

backend_cors_allowed_origin = "https://*"

# 1. BCD - The Start
# US runs at minute 00, EU runs at minute 30
bcd_region_schedules = {
  "us-central1"  = "0 0,6,12,18 * * *"
  "europe-west1" = "30 0,6,12,18 * * *"
}

# 2. Web Features - 1 Hour Later
# Needs the BCD data first for Browser Releases
# (Hours shifted to 1, 7, 13, 19)
web_features_region_schedules = {
  "us-central1"  = "0 1,7,13,19 * * *"
  "europe-west1" = "30 1,7,13,19 * * *"
}

# 3. Signals / Chromium  / Web Features Mapping - 2 Hours Later
# These need the Web Features data first
# (Hours shifted to 2, 8, 14, 20)
developer_signals_region_schedules = {
  "us-central1"  = "0 2,8,14,20 * * *"
  "europe-west1" = "30 2,8,14,20 * * *"
}

chromium_region_schedules = {
  "us-central1"  = "0 2,8,14,20 * * *"
  "europe-west1" = "30 2,8,14,20 * * *"
}

web_features_mapping_region_schedules = {
  "us-central1"  = "0 3,9,15,21 * * *"
  "europe-west1" = "30 3,9,15,21 * * *"
}

# 4. UMA - 3 Hours Later
# UMA needs the Chromium data first
# (Hours shifted to 3, 9, 15, 21)
uma_region_schedules = {
  "us-central1"  = "0 3,9,15,21 * * *"
  "europe-west1" = "30 3,9,15,21 * * *"
}

# 5. WPT - Takes a while. It runs once daily.
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
notification_channel_ids    = ["projects/web-compass-prod/notificationChannels/4991947607216940054"]
email_service_account_email = "emailer-job-prod@webstatus-dev-internal-prod.iam.gserviceaccount.com"

chime_details = {
  env          = "prod"
  bcc          = "webstatus-dev-mail-log-prod@google.com"
  from_address = "noreply-webstatus-dev@google.com"
}
