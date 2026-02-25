# Playwright Execution and Debugging

## Rapid Iteration (`SKIP_FRESH_ENV`)

If you already have a local environment running (via `make start-local`, `make port-forward-manual`, `make dev_fake_users`, and `make dev_fake_data`), you can skip the environment teardown and setup to run tests faster.

**Command**: `SKIP_FRESH_ENV=1 make playwright-test`

_Note: Use this only when your local environment is in a known good state._

## Debugging GitHub Failures

When E2E tests fail in CI, you can download the traces and reports locally for analysis.

1. **Find the Run ID**: `make print-gh-runs`
2. **Download the Report**: `make download-playwright-report-from-run-<RUN_ID>`
3. **Open the Report**: `make playwright-open-report`
4. **Inspect Traces**:
   - List traces: `make playwright-show-traces`
   - Open a specific trace: `npx playwright show-trace playwright-report/data/<trace_hash>.zip`

## Local Interactive Tools

- **Playwright UI**: `make playwright-ui` (best for seeing tests run live and inspecting locators).
- **Step-through Debugging**: `make playwright-debug`.
