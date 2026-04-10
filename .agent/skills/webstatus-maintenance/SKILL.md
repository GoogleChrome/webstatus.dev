---
name: webstatus-maintenance
description: Use when upgrading toolchain versions (Go, Node.js, Terraform, Playwright), updating the DevContainer, Nix environment, or Github CI configurations.
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
5. `flake.nix` (If pinning a specific `nixpkgs` commit for Go)

After updating the files, you should run `make go-update && make go-tidy` to ensure the `go.mod` dependencies are compatible with the new version.

### Upgrading Node.js

The Node.js version must be kept in sync across:

1. `.devcontainer/devcontainer.json` (`features -> node -> version`)
2. `.devcontainer/Dockerfile` (`FROM mcr.microsoft.com/devcontainers/typescript-node:X-22`)
3. `.github/workflows/ci.yml` (`NODE_VERSION` environment variable)
4. `.github/workflows/devcontainer.yml` (`NODE_VERSION` environment variable)
5. `images/nodejs_service.Dockerfile` (`FROM node:X.Y.Z-alpine...`)
6. `flake.nix` (If pinning a specific `nixpkgs` commit for Node.js)

After updating, run `make node-update` and test the frontend build.

### Upgrading Playwright

Playwright requires its NPM package and OS-level dependencies to stay in sync:

1. Update `playwright` and `@web/test-runner-playwright` in `frontend/package.json`.
2. Update the system dependencies in `.github/workflows/ci.yml` (the `npx playwright install --with-deps` step).
3. On Nix: Playwright browsers are cached in `.nix/browsers` and patched automatically by the shell hook in `flake.nix`.

### Upgrading via Nix

The Nix environment provides an alternative toolchain.

- **Bumping All Tools**: Run `nix flake update` to update all tools to their latest versions in `nixpkgs-unstable`.
- **Pinning Versions**: To pin a specific tool version, update `flake.nix` to use a specific `nixpkgs` commit hash (see `docs/nix-setup.md`).

### Upgrading Go-based Tools

Tools like `wrench`, `oapi-codegen`, and `golines` are managed via `tools/go.mod`.

- To upgrade a tool, navigate to the `tools/` directory and run `go get <package>@<version>`.
- Run `go mod tidy` in the `tools/` directory.

### Upgrading DevContainer Features

Other DevContainer tools (Skaffold, Shellcheck, GitHub CLI) are managed within `.devcontainer/devcontainer.json`.

- Find the relevant feature under the `features` object and update its `"version"`.
- If modifying Terraform, also ensure `infra/providers.tf` (`required_version`) and CI checks match the new version.

## Documentation Updates

If you change how versions are managed or introduce a new critical dependency:

- Update `docs/maintenance.md` to reflect the new update path.
- Update `docs/nix-setup.md` if the Nix environment logic changes.
- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
