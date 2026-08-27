---
name: webstatus-e2e
description: Use when writing, modifying, or debugging Playwright end-to-end (E2E) tests for webstatus.dev.
---

# webstatus-e2e

This skill provides guidance for working with the End-to-End (E2E) test suite in `webstatus.dev`, which is built using Playwright and TypeScript.

## Architecture & Location

The E2E test suite is organized into three distinct suites based on scope and execution speed:

- **Synthetic Smoke Probes (`e2e/synthetic/`)**:
  - **Purpose**: Ultra-lightweight cluster canary probes (~15s execution) checking live frontend routing, search box interactivity, and backend health.
  - **Execution**: `make playwright-synthetic`
- **Visual Regression Tests (`e2e/visual/`)**:
  - **Purpose**: Dual-theme dark/light rendering, layout shift, dialog states, and multi-page pagination testing against deterministic mock fixtures (`setupVisualFixtures`).
  - **Execution**: `make playwright-visual` (update with `make playwright-update-snapshots`)
- **Functional User Journeys (`e2e/functional/`)**:
  - **Purpose**: Stateful end-to-end user workflows against live cluster emulators (Cloud Spanner, Datastore, Valkey, Wiremock).
  - **Execution**: `make playwright-functional`

- **Configuration**: `playwright.config.ts` handles browser definitions, retries, and worker limits.

## Architecture

For a detailed technical guide on the local development environment (Skaffold/Minikube), data population strategies, and the CI/PR validation lifecycle, see [references/architecture.md](references/architecture.md).

## Guidelines (Do's and Don'ts)

- **DO** cross-reference all code against the official Google TypeScript Style Guide. If you are unsure about a specific style rule, DO NOT assume; you MUST ask the user for clarification.
- **DO** add E2E tests for critical user journeys (e.g., login flows, complex search operations, saving a search).
- **DON'T** write E2E tests for small component-level interactions; those belong in frontend unit tests (`frontend/src/**/*.test.ts`).
- **DO** use resilient locators. Prefer using `data-testid` attributes (e.g., `page.getByTestId('submit-btn')`) over brittle CSS classes or XPath.
- **DO** move the mouse to a neutral position (e.g., `page.mouse.move(0, 0)`) before taking visual snapshots to avoid flaky tests caused by unintended hover effects on UI elements.
- **DO** use **Wiremock** (available at `localhost:8080` via port-forward) to mock GitHub API responses, such as user profiles and email lookups during login.
- **DO** use `waitForChartCompletion` and `waitForTabbedChartCompletion` hooks (from `utils.ts`) for Google Charts instead of naive `.waitForSelector` to avoid timeout races.
- **DO** use `toBeAttached()` instead of `toBeVisible()` to cleanly bypass WebKit strict-mode 0px bounding box quirks for inline host elements or absolutely positioned fragments.
- **DO** explicitly use `await` on asynchronous Playwright matchers like `toBeChecked()` to prevent tests from skipping past Lit hydration cycles synchronously.
- **DO** validate downloaded files (e.g., CSV exports) **structurally** in functional tests by asserting headers, filename, and row counts (`rowCount > 1`), rather than using brittle byte-for-byte `toMatchSnapshot` assertions that break on database seed changes. Exact formatting snapshots belong in visual tests against static fixtures.

## Configuration & Stability

- **Single Worker**: Tests operate on stateful emulator accounts. To ensure database isolation, `workers: 1` is enforced in `playwright.config.ts`.
- **Matrix Sharding**: In CI, functional tests are sharded 3-way (`1/3`, `2/3`, `3/3`) to distribute execution evenly across parallel runners.
- **Retries**: Playwright tests are configured to retry twice on failure only when running in CI (`CI=true`).
- **Browsers**: Tested across Chromium, Firefox, and WebKit.

## Execution & Debugging

- For detailed instructions on rapid iteration, debugging CI failures, and using traces, see [references/execution-and-debugging.md](references/execution-and-debugging.md).

## Commands Summary

- Use the `Makefile` in the project root:
  - `make playwright-synthetic`: Runs only the fast canary smoke probes.
  - `make playwright-visual`: Runs visual regression snapshots against the cluster.
  - `make playwright-functional`: Runs stateful functional user workflows.
  - `make playwright-test`: Runs all Playwright test suites.
  - `SKIP_FRESH_ENV=1 make playwright-test`: Rapidly iterates on tests by skipping cluster provisioning (requires existing running environment).
  - `make playwright-ui`: Runs tests in Playwright's interactive UI mode.
  - `make playwright-debug`: Runs tests in debug mode with inspector.
  - `make playwright-update-snapshots`: Updates visual regression snapshots.

## Documentation Updates

When modifying playwright configuration, retries, or execution strategies:

- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
- Update these very skills files if you introduce new established patterns for testing.
