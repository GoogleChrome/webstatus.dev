# Infra Readme

## One time setup

This section includes one time setup instructions for the GCP projects

### Step 1: Set environment variables

For Staging:

```sh
HOST_VPC_PROJECT=web-compass-staging
INTERNAL_PROJECT=webstatus-dev-internal-staging
PUBLIC_PROJECT=webstatus-dev-public-staging
```

For Prod:

```sh
#TODO
```

```sh
gcloud --project=${INTERNAL_PROJECT} services enable compute.googleapis.com
gcloud --project=${PUBLIC_PROJECT} services enable compute.googleapis.com
```

In the UI, follow the
[instructions](https://cloud.google.com/vpc/docs/provisioning-shared-vpc#set-up-shared-vpc)
to setup a shared vpc in ${HOST_VPC_PROJECT} and attach ${INTERNAL_PROJECT} and
${PUBLIC_PROJECT}.


