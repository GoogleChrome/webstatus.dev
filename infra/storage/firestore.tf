resource "google_firestore_database" "firestore_db" {
  # project                     = data.google_project.project.id
  name        = "${var.env_id}-db"
  location_id = "us-east1"
  type        = "DATASTORE_MODE"
  # concurrency_mode            = "OPTIMISTIC"
  # app_engine_integration_mode = "DISABLED"

}