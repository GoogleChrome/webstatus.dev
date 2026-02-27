# Maintenance & Infrastructure Architecture

This document provides a guide to the repository's toolchain synchronization and infrastructure maintenance procedures.

## 1. Synchronized Toolchain Upgrades

To prevent environment drift, core tools must be updated across all environments (DevContainer, CI, and Production) simultaneously.

### Go Upgrades

- **DevContainer**: [`.devcontainer/devcontainer.json`](../../../.devcontainer/devcontainer.json)
- **CI Pipelines**: [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml), [`.github/workflows/devcontainer.yml`](../../../.github/workflows/devcontainer.yml)
- **Docker Images**: [`images/go_service.Dockerfile`](../../../images/go_service.Dockerfile)

### Node.js Upgrades

- **DevContainer**: [`.devcontainer/devcontainer.json`](../../../.devcontainer/devcontainer.json)
- **CI Pipelines**: [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml)
- **Docker Images**: [`images/nodejs_service.Dockerfile`](../../../images/nodejs_service.Dockerfile)

## 2. Infrastructure Configuration

The repository uses **Terraform** for Cloud Run, Spanner, and Valkey provisioning.

- **Workflows**: Defined in [`infra/ingestion/workflows.tf`](../../../infra/ingestion/workflows.tf).
- **Service Configuration**: Terraform states are managed centrally; local changes should be validated against the CI plan before merge.

## 3. Maintenance Procedures

For a full list of maintenance commands and update paths for Playwright and other features, refer to the high-level [Repository Maintenance Guide](../../../docs/maintenance.md).
