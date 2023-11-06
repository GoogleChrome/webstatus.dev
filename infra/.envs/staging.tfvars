projects = {
    host = "web-compass-staging"
    internal = "webstatus-dev-internal-staging"
    public = "webstatus-dev-public-staging"
}
project_name="web-compass-staging"
spanner_processing_units=200
deletion_protection=true
secret_ids={
    github_token = "jamestestGHtoken-readonly"
}
datastore_region_id = "nam5"
region_information= {
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