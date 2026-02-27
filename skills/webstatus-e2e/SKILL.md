---
name: webstatus-e2e
description: Use when writing, modifying, or debugging Playwright end-to-end (E2E) tests for webstatus.dev.
---

# webstatus-e2e

This skill provides guidance for working with the End-to-End (E2E) test suite in `webstatus.dev`, which is built using Playwright and TypeScript.

## Architecture & Location

- **Directory**: The E2E tests are located in the `e2e/` directory.
- **Framework**: **Playwright** with **TypeScript**.
- **Configuration**: `playwright.config.ts` handles browser definitions, retries, and worker limits.

## Architecture

For a detailed technical guide on the local development environment (Skaffold/Minikube), data population strategies, and the CI/PR validation lifecycle, see [references/architecture.md](references/architecture.md).

## Guidelines (Do's and Don'ts)

- **DO** add E2E tests for critical user journeys (e.g., login flows, complex search operations, saving a search).
- **DON'T** write E2E tests for small component-level interactions; those belong in frontend unit tests (`frontend/src/**/*.test.ts`).
- **DO** use resilient locators. Prefer using `data-testid` attributes (e.g., `page.getByTestId('submit-btn')`) over brittle CSS classes or XPath.
- **DO** move the mouse to a neutral position (e.g., `page.mouse.move(0, 0)`) before taking visual snapshots to avoid flaky tests caused by unintended hover effects on UI elements.
- **DO** use **Wiremock** (available at `localhost:8080` via port-forward) to mock GitHub API responses, such as user profiles and email lookups during login.

## Configuration & Stability

- **Single Worker**: Tests currently operate on the same end-user accounts, which means they can interfere with each other if run concurrently. To ensure stability, `workers: 1` is strictly enforced in `playwright.config.ts`.
- **Retries**: Playwright tests are configured to retry twice on failure only when running in a CI environment. If you want to simulate this locally and test flakiness, you can prefix your command with `CI=true` (e.g., `CI=true make playwright-test`).
- **Browsers**: If you ever need to test against new browsers (e.g., mobile viewports, branded Edge/Chrome), modify the `projects` array within `playwright.config.ts`.

## Execution & Debugging

- For detailed instructions on rapid iteration, debugging CI failures, and using traces, see [references/execution-and-debugging.md](references/execution-and-debugging.md).

## Commands Summary

- Use the `Makefile` in the project root:
  - `make playwright-test`: Sets up a fresh local environment and runs the test suite.
  - `SKIP_FRESH_ENV=1 make playwright-test`: Rapidly iterates on E2E tests by skipping the full Skaffold/Minikube setup (requires an already running environment).
  - `make playwright-ui`: Runs the tests in Playwright's interactive UI mode.
  - `make playwright-debug`: Runs the tests in debug mode.
  - `make playwright-update-snapshots`: Updates visual regression snapshots.

## Documentation Updates

When modifying playwright configuration, retries, or execution strategies:

- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
- Update these very skills files if you introduce new established patterns for testing.
