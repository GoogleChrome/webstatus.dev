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

backend_domains   = ["api-webstatus-dev.corp.goog"]
frontend_domains  = ["website-webstatus-dev.corp.goog"]
frontend_base_url = "https://website-webstatus-dev.corp.goog"

# Temporary for UbP.
custom_ssl_certificates_for_frontend = ["ub-self-sign"]
custom_ssl_certificates_for_backend  = ["ub-self-sign"]

backend_cache_settings = {
  default_duration                  = "1h"
  aggregated_feature_stats_duration = "24h"
}

# Needed for UbP.
backend_cors_allowed_origin = "https://website-webstatus-dev.corp.goog"

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


firebase_api_key_location = "staging-firebase-app-api-key"

auth_github_config_locations = {
  client_id     = "staging-github-client-id"
  client_secret = "staging-github-client-secret"
}

# TODO: Once staging is public, we should change the minimum instance count to
# match production.
backend_min_instance_count  = 0
frontend_min_instance_count = 0
notification_channel_ids    = ["projects/web-compass-staging/notificationChannels/7136127183667686021"]
email_service_account_email = "emailer-job-staging@webstatus-dev-internal-staging.iam.gserviceaccount.com"
