# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

locals {
  gh_client_id     = sensitive(try(data.google_secret_manager_secret_version_access.gh_client_id[0].secret_data, ""))
  gh_client_secret = sensitive(try(data.google_secret_manager_secret_version_access.gh_client_secret[0].secret_data, ""))
}

data "google_secret_manager_secret_version_access" "gh_client_id" {
  count    = var.github_config_locations.client_id != null ? 1 : 0
  provider = google.internal_project
  secret   = var.github_config_locations.client_id
}

data "google_secret_manager_secret_version_access" "gh_client_secret" {
  count    = var.github_config_locations.client_secret != null ? 1 : 0
  provider = google.internal_project
  secret   = var.github_config_locations.client_secret
}

resource "google_identity_platform_tenant" "tenant" {
  provider                 = google.internal_project
  display_name             = var.env_id
  allow_password_signup    = false
  enable_email_link_signin = false
}

resource "google_identity_platform_tenant_default_supported_idp_config" "github_idp_config" {
  count         = local.gh_client_secret != "" ? 1 : 0
  provider      = google.internal_project
  enabled       = true
  tenant        = google_identity_platform_tenant.tenant.name
  idp_id        = "github.com"
  client_id     = local.gh_client_id
  client_secret = local.gh_client_secret
}
