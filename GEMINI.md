# Gemini Code Assist Configuration for webstatus.dev

<!-- Last analyzed commit: c3beadb0b4f8b7b6e38c089aa770a2123b4ee017 -->

This document provides context to Gemini Code Assist to help it generate more accurate and project-specific code suggestions.

## 1. Project Overview

`webstatus.dev` tracks the status and implementation of web platform features across major browsers. It aggregates data from various sources, including Web Platform Tests (WPT), Browser-Compat-Data (BCD), and Chromium's UMA (Usage Metrics). The architecture consists of a Go API (running as a Cloud Run service), multiple Go jobs (running as Cloud Run jobs), and a TypeScript frontend (running as a Cloud Run service). It utilizes Spanner for the database and Valkey as a cache layer.

## 2. Local Development Workflow

This section describes the tools and commands for local development.

- **Devcontainer**: The project uses a devcontainer for a consistent development environment.
- **Skaffold & Minikube**: Local development is managed by `skaffold`, which deploys services to a local `minikube` Kubernetes cluster.
- **Makefile**: Common development tasks are scripted in the `Makefile`. See below for key commands.

### 2.1. Key Makefile Commands

- **`make start-local`**: Starts the complete local development environment using Skaffold and Minikube. This includes live-reloading for code changes.
- **`make port-forward-manual`**: After starting the environment, run this to expose the services (frontend, backend, etc.) on `localhost`.
- **`make test`**: Runs the Go and TypeScript unit tests. Use `make go-test` to run only Go tests.
- **`make precommit`**: Runs a comprehensive suite of checks including tests, linting (`golangci-lint` configured via `.golangci.yaml`), and license header verification. This is the main command to run before submitting a pull request. Common linting errors to watch for include `exhaustruct` (missing struct fields) and `nlreturn` (missing newline before return).
- **`make gen`**: Regenerates all auto-generated code (from OpenAPI, JSON Schema, ANTLR). Use `make openapi` for just OpenAPI changes.
- **`make dev_workflows`**: Populates the local Spanner database by running the data ingestion jobs against live data sources.
- **`make dev_fake_data`**: Populates the local Spanner database with a consistent set of fake data for testing.
- **`make spanner_new_migration`**: Creates a new Spanner database migration file in `infra/storage/spanner/migrations`.

## 3. Specialized Skills

Detailed architectural guidance, coding standards, and "how-to" guides for specific domains have been moved to **Gemini Skills**.

To activate these rules in your session, run:

```bash
make link-skills
```

The available skills are:

- `webstatus-backend`: Go API, Spanner mappers, and OpenAPI.
- `webstatus-frontend`: Lit web components and component testing.
- `webstatus-e2e`: Playwright E2E testing and debugging.
- `webstatus-ingestion`: Scheduled data ingestion workflows.
- `webstatus-workers`: Pub/Sub notification pipeline.
- `webstatus-search-grammar`: ANTLR search query parsing.

## 4. Updating the Knowledge Base

To keep the skills and this document up-to-date, you can ask me to analyze the latest commits and update my knowledge base. I will use the hidden marker at the end of this file to find the commits that have been made since my last analysis.

### 4.1. Prompt for Updating

You can use the following prompt to ask me to update my knowledge base:

> Please update your knowledge base by analyzing the commits since the last analyzed commit stored in `GEMINI.md`.

### 4.2. Process

When you give me this prompt, I will:

1.  Read the `GEMINI.md` file to find the last analyzed commit SHA.
2.  Use `git log` to find all the commits that have been made since that SHA.
3.  Analyze the new commits, applying the "Verify, Don't Assume" principle by consulting relevant sources of truth (e.g., `openapi.yaml` for API changes, migration files for schema changes). Use the `get_pull_request` and `get_pull_request_comments` tools to get the pull request information for **every relevant PR** found in the commits. Read the comments to understand the context and architectural decisions.
4.  Update the relevant Skill files in `skills/` first.
5.  Update the last analyzed commit SHA near the top of this file only after all other updates are complete.
