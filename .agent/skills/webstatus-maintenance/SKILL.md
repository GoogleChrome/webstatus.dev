---
name: webstatus-maintenance
description: Use when upgrading toolchain versions (Go, Node.js, Terraform, Playwright) or updating the DevContainer and Github CI configurations.
---

# webstatus-maintenance

This skill provides guidance for updating the core toolchain versions across the `webstatus.dev` repository.

## Architecture

For a technical overview of synchronized upgrades and infrastructure maintenance, see [references/architecture.md](references/architecture.md).

## General Guidelines

- **DO** cross-reference all infrastructure/scripts against the official Google Style Guides (e.g. for Shell Scripts, YAML). If you are unsure about a specific style rule, DO NOT assume; you MUST ask the user for clarification.

## Synchronized Upgrades

When asked to update a specific language or tool, you must update it in all of its respective locations to prevent environment drift.

### Upgrading Go

The Go version must be kept in sync across:

1. `.devcontainer/devcontainer.json` (`features -> go -> version`)
2. `.github/workflows/ci.yml` (`GO_VERSION` environment variable)
3. `.github/workflows/devcontainer.yml` (`GO_VERSION` environment variable)
4. `images/go_service.Dockerfile` (`FROM golang:X.Y.Z-alpine...`)

After updating the files, you should run `make go-update && make go-tidy` to ensure the `go.mod` dependencies are compatible with the new version.

### Upgrading Go Workspace & Tools Dependencies (`tools/go.mod` vs `go.work`)

Because `webstatus.dev` uses a multi-module workspace (`go.work`), `tools/go.mod` (`oapi-codegen`, `wrench`, `addlicense`) is included right inside the active workspace when `make go-workspace-setup` runs:

- **Synchronized `tools` Upgrades**: When upgrading shared libraries (like `github.com/getkin/kin-openapi` or `runtime`), you **MUST** also update `tools/go.mod` (`github.com/oapi-codegen/oapi-codegen/v2`) to a compatible version. Otherwise, Go's Minimal Version Selection (MVS) will compile `tools` against the upgraded `kin-openapi` models and cause type mismatch errors (`mismatched types openapi3.MappingRef and string`).
- **Makefile `@latest` Flags**: Ensure targets like `go-update-tools` use `go get -u ...@latest` for all tools so they do not lag behind when running updates.
- **`golangci-lint` Configuration**: When bumping `golangci-lint` versions across v2 config format, ensure `goconst.ignore-tests: true` is preserved so table-driven test fixtures are not flagged. Note that we do NOT use brittle `ignore-string-values` whitelists for Spanner query parameter keys (`startAt`, `pageSize`, etc.); instead, we exclude `goconst` specifically for `lib/gcpspanner/.*\.go` under `linters.exclusions.rules` (`path: lib/gcpspanner/.*\.go`), because parameter keys naturally repeat across dozens of independent query methods matching raw SQL `@param` bindings.

### Upgrading Node.js

The Node.js version must be kept in sync across:

1. `.devcontainer/devcontainer.json` (`features -> node -> version`)
2. `.devcontainer/Dockerfile` (`FROM mcr.microsoft.com/devcontainers/typescript-node:X-22`)
3. `.github/workflows/ci.yml` (`NODE_VERSION` environment variable)
4. `.github/workflows/devcontainer.yml` (`NODE_VERSION` environment variable)
5. `images/nodejs_service.Dockerfile` (`FROM node:X.Y.Z-alpine...`)

After updating, run `make node-update` and test the frontend build.

### Upgrading Playwright

Playwright requires its NPM package and OS-level dependencies to stay in sync:

1. Update `playwright` and `@web/test-runner-playwright` in `frontend/package.json`.
2. Update the system dependencies in `.github/workflows/ci.yml` (the `npx playwright install --with-deps` step).

### Upgrading DevContainer Features

Other DevContainer tools (Terraform, Skaffold, Shellcheck, GitHub CLI) are managed within `.devcontainer/devcontainer.json`.

- Find the relevant feature under the `features` object and update its `"version"`.
- If modifying Terraform, also ensure the `.terraform-version` file (if one exists) or CI checks match the new version.
- **Lockfile Synchronization (`devcontainer-lock.json`)**: `.devcontainer/devcontainer-lock.json` is committed to version control to guarantee deterministic container builds across developers and CI (`.github/workflows/devcontainer.yml`) by locking exact OCI feature digests (`@sha256:...`) and resolved versions. Whenever you modify features or version constraints in `.devcontainer/devcontainer.json`, you **MUST** ensure `.devcontainer/devcontainer-lock.json` is updated and committed alongside `devcontainer.json`.
  - To regenerate or update the lockfile locally, run `devcontainer up` / `devcontainer build` or `devcontainer features upgrade --workspace-folder .` using the DevContainer CLI.
  - Always stage both files: `git add .devcontainer/devcontainer.json .devcontainer/devcontainer-lock.json`.

## Documentation Updates

If you change how versions are managed or introduce a new critical dependency:

- Update `docs/maintenance.md` to reflect the new update path.
- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
