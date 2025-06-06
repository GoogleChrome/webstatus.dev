/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {playwrightLauncher} from '@web/test-runner-playwright';
const filteredLogs = [
  'Running in dev mode',
  'Lit is in dev mode',
  // sl-tree-item has its own reactivity that we cannot control. Ignore for now.
  'Element sl-tree-item scheduled an update',
  // From the date range picker
  'WebstatusFormDateRangePicker: minimumDate, maximumDate, startDate, and endDate are required properties.',
  // Unknown errors print to console for users to send us.
  'Saved Search Unknown Test Error',
  'User Saved Searches Unknown Test Error',
];

export default /** @type {import("@web/test-runner").TestRunnerConfig} */ ({
  concurrency: 4,
  /** Resolve bare module imports */
  nodeResolve: {
    exportConditions: ['browser', 'development'],
  },

  // in a monorepo you need to set the root dir to resolve modules
  rootDir: '.',

  files: [
    // Have to compile tests
    // Taken from https://github.com/open-wc/create/blob/master/src/generators/testing-wtr-ts/templates/static/web-test-runner.config.mjs
    'build/**/test/*.test.js',
  ],
  testRunnerHtml: testFramework => `
  <html>
    <body>
      <script type="module" src="${testFramework}"></script>
      <script type="module">
        import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
        setBasePath('/public/img/shoelace');
      </script>
    </body>
  </html>
  `,

  /** Filter out lit dev mode logs */
  filterBrowserLogs(log) {
    for (const arg of log.args) {
      if (typeof arg === 'string' && filteredLogs.some(l => arg.includes(l))) {
        return false;
      }
    }
    return true;
  },

  // How long a test file can take to finish.
  testsFinishTimeout: 1000 * 60 * 1, // (1 min)
  // mocha config https://mochajs.org/api/mocha
  testFramework: {config: {timeout: 30000}},

  browsers: [
    playwrightLauncher({
      product: 'chromium',
      // For some we need to limit the concurrency for Chromium to ensure test stability.
      concurrency: 1,
    }),
    playwrightLauncher({
      product: 'firefox',
      // For some we need to limit the concurrency for Firefox to ensure test stability.
      concurrency: 1,
    }),
    playwrightLauncher({
      product: 'webkit',
      // For some we need to limit the concurrency for Safari to ensure test stability.
      concurrency: 1,
    }),
  ],
});
