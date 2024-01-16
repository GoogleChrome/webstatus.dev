# Creating an Ingestion Workflow

There are multiple steps to creating an ingestion workflow. This document serves
as a how-to guide to create a new workflow.

## Step 0. Background

The current tech stack for workflows consists of using
[GCP Workflows](https://cloud.google.com/workflows/docs/overview). This allows
the team to not have to handle orchestration between multiple steps manually.
All the team has to do is create a workflow yaml with the various steps.
Workflows provide a
[standard library](https://cloud.google.com/workflows/docs/reference/stdlib/overview)
that can be used in steps out of the box. Additionally, workflows provides a
[set of connectors](https://cloud.google.com/workflows/docs/reference/googleapis#list_of_supported_connectors)
that allow workflows to talk to various Google Cloud products out of the box.
Together, the standard library and connectors may remove the need to write
custom code. It is important to be aware of the
[limits of workflows](https://cloud.google.com/workflows/quotas) when designing
a new one.

If custom code is not needed, jump to step 2.

# Step 1. Build the Custom Code Service

Follow the [directions](creating-a-new-service.md) to create a new service

# Step 2. Create the workflow file

TODO
