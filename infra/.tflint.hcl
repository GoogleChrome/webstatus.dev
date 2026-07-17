// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

plugin "terraform" {
  enabled = true
  preset  = "all"
  version = "0.15.0"
  source  = "github.com/terraform-linters/tflint-ruleset-terraform"
  # Workaround for https://github.com/terraform-linters/tflint/issues/2591
  # We still want to do some verification even if it is the legacy way.
  # That is better than the alternatives of 1) not verifying at all or 2) downgrading to an older version.
  signature = "pgp"
}

plugin "google" {
  enabled = true
  version = "0.39.0"
  source  = "github.com/terraform-linters/tflint-ruleset-google"
  # Workaround for https://github.com/terraform-linters/tflint/issues/2591
  # We still want to do some verification even if it is the legacy way.
  # That is better than the alternatives of 1) not verifying at all or 2) downgrading to an older version.
  signature = "pgp"
}
