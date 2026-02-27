# How to Add a New Scheduled Workflow

This guide outlines the process for adding a new scheduled data ingestion workflow. Workflows run as scheduled Kubernetes/Cloud Run **Jobs** (unlike workers which are long-running pods).

## 1. Create the Workflow Application

- Create a new directory in `workflows/steps/services/<new_workflow>`.
- Implement `main.go`.
- Implement data fetching and parsing logic in a `pkg/data` subdirectory (`downloader.go`, `parser.go`).
- Add a `manifests/job.yaml` file for the local Kubernetes Job definition.
- Add a `skaffold.yaml` file.

## 2. Update the `Makefile`

- Add a new target to the `make dev_workflows` command in the root `Makefile` to allow running the new job locally.

## 3. Terraform Integration

Workflows are deployed as scheduled Cloud Run Jobs via Cloud Scheduler.

1. **Add the Module**: In [`infra/ingestion/workflows.tf`](../../../infra/ingestion/workflows.tf), add a new `module "workflow"` block for your new job. This defines the Cloud Run Job resource.
2. **Configure Scheduling**:
   - In [`infra/variables.tf`](../../../infra/variables.tf), add a new variable for your workflow's schedule (e.g., `my_new_workflow_region_schedules`).
   - In [`infra/.envs/staging.tfvars`](../../../infra/.envs/staging.tfvars) and [`infra/.envs/prod.tfvars`](../../../infra/.envs/prod.tfvars), add the new variable and set appropriate cron schedules.
   - In [`infra/main.tf`](../../../infra/main.tf), pass your new schedule variable into the `module "ingestion"` block.
   - In [`infra/ingestion/main.tf`](../../../infra/ingestion/main.tf), add the new variable to the module's inputs.
   - In [`infra/ingestion/workflows.tf`](../../../infra/ingestion/workflows.tf), pass the `region_schedules` variable to your new workflow module.

## 4. Pull Requests

For new workflows, split your work into multiple PRs:

1. **Data Layer PR**: Schema migration, new types, `gcpspanner` mapper, and client methods.
2. **Workflow Logic PR**: The consumer implementation, including its processor, parser, and downloader.
3. **Infrastructure PR**: Terraform changes to deploy the new Cloud Run Job.
