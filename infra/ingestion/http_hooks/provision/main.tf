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