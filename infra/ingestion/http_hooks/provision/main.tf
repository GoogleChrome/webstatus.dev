# Copyright 2023 Google LLC
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

module "gcloud" {
  source  = "terraform-google-modules/gcloud/google"
  version = "3.1.2"

  platform = "linux"

  create_cmd_entrypoint = "gcloud"
  create_cmd_body       = <<EOT
--project=${var.project_name} beta tasks queues create ${var.task_name} \
--http-uri-override=scheme:https,host:workflowexecutions.googleapis.com,path:/v1/projects/${var.project_name}/locations/${var.region}/workflows/${var.workflow_name}/executions \
--http-method-override=POST \
--location=${var.region}
EOT
}