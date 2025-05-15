# Testing

_This doucument assumes you are using the devcontainer._

## Table Of Contents

1. Run Precommit Suite

   - Go Tests
   - TypeScript Tests
   - License Check
     - License Fix
   - Linting
     - Attempt Automatic Lint Error Fixing

1. Run Playwright Tests

   - Update Screenshots
   - Download GitHub Playwright Test Results
   - Analyze Playwright Test Results
     - Show Playwright Test Report
     - Show Playwright Test Trace

## Run Precommit Suite

The `precommit` command is a wrapper command that includes many tools (tests minus Playwright tests, license check,
linting, etc).

Run the following command to run the precommit suite: `make precommit`.

Below describes some of the commands from the `precommit` command that a developer can run.

### Go Tests

To run only the Go tests, run: `make go-test`.

### TypeScript Tests

To run only the TypeScript tests (excluding Playwright tests), run: `make node-test`.

### License Check

Run the following command to check the license headers for all code: `make license-check`.

#### License Fix

Run the following command to automatically fix the license errors: `make license-fix`.

### Linting

Run the following command to lint all the code: `make lint`.

#### Attempt Automatic Lint Error Fixing

Some lint errors can be fixed automatically. This command will try to fix all the lint errors: `make lint-fix`.

Developers may still need to inspect and manually fix the error.

## Playwright Tests

### Note about running Playwright Makefile Targets

If you have a running local development environment
(established via `make start-local`) with the necessary port forwarding
(`make port-forward-manual`), fake users (`make dev_fake_users`) and
fake data (`make dev_fake_data`), you can expedite Playwright testing by
skipping the environment setup.

To do this, prefix your `make` command with `SKIP_FRESH_ENV=1`. This is
particularly useful for rapid iteration, as `make start-local` leverages
Skaffold to rebuild the application on file saves.

For example: `make playwright-test` becomes `SKIP_FRESH_ENV=1 make playwright-test`.

**Important:** Skipping the environment setup assumes your existing local
environment is in a known good state. If inconsistencies arise, ensure you run
the Playwright target without `SKIP_FRESH_ENV=1` to create a clean environment.
This is the default behavior to guarantee a reliable testing environment.

### Running Playwright Tests

Run the following command to run the Playwright tests: `make playwright-test`.

### Update Screenshots

There are some tests that check screenshots. That means, your changes may impact the existing screenshots. Run the
following command to update the screenshots: `make playwright-update-snapshots`.

### Download GitHub Playwright Test Results

Sometimes the Playwright tests fail in GitHub and you will want to download the results and analyze them.

1. Find the GitHub Action ID by running: `make print-gh-runs`
   - Example output:
   ```
   STATUS  TITLE     WORKFLOW  BRANCH             EVENT        ID          ELAPSED  AGE
   ✓       PR title  build     jcscottiii/branch  pull_request 9551121552  12m49s   about 2 hours ago
   X       PR title  build     jcscottiii/branch  pull_request 9551110554  16m28s   about 2 hours ago
   ```
   In this example, we will use ID `9551110554`
2. Download the report: `make download-playwright-report-from-run-$RUNID`
   - In this example, you would run: `make download-playwright-report-from-run-9551110554`

### Analyze Playwright Test Results

This section describes how to analyze the Playwright tests. The following sections expect there is a `playwright-report`
directory that is present by:

- running the Playwright tests locally, or
- downloading the report from GitHub.

#### Show Playwright Test Report

Given the `playwright-report` directory exists, run the following command: `make playwright-open-report` to open the [report](https://playwright.dev/docs/trace-viewer-intro#opening-the-html-report).

#### Show Playwright Test Trace

1. Given there are failures, there should be traces. You can list each failure trace by running: `make playwright-show-traces`
   - Example:
   ```
   playwright-report/data/9bd87f9fe48efeb348ad4c6f05d01743b15501b8.zip
   playwright-report/data/4dc5b7341597474f7fb1abedf1b2017e3b271eef.zip
   playwright-report/data/0baeb6be37c91cd90156df684dc28a7b51970df3.zip
   ```
2. Open the [trace](https://playwright.dev/docs/trace-viewer#opening-the-trace): `npx playwright show-trace path/to/trace.zip`
   - Example: `npx playwright show-trace playwright-report/data/9bd87f9fe48efeb348ad4c6f05d01743b15501b8.zip`
