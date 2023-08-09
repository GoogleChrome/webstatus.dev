resource "google_spanner_instance" "main" {
  name             = "${var.env_id}-spanner"
  config           = var.spanner_region_id
  display_name     = "${var.env_id} Spanner"
  processing_units = var.spanner_processing_units
  force_destroy    = !var.deletion_protection
}

resource "google_spanner_database" "database" {
  instance                 = google_spanner_instance.main.name
  name                     = "${var.env_id}-database"
  version_retention_period = "3d"
  deletion_protection      = var.deletion_protection
}