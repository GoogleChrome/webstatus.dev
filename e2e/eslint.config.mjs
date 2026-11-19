/**
 * Copyright 2026 Google LLC
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

import {defineConfig} from 'eslint/config';
import tseslint from 'typescript-eslint';

// Lint config for the Playwright E2E tests.
//
// This is intentionally narrow: it only catches missing `await`s on async
// Playwright APIs (web-first assertions like `toBeVisible()`, route handlers,
// `test.step()`, etc.). An un-awaited assertion is not reliably part of the
// test result. See https://playwright.dev/docs/best-practices#lint-your-tests
export default defineConfig([
  {
    ignores: ['test-results/**'],
  },
  {
    files: ['**/*.ts'],
    languageOptions: {
      parser: tseslint.parser,
      parserOptions: {
        // Type information comes from e2e/tsconfig.json.
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      '@typescript-eslint': tseslint.plugin,
    },
    rules: {
      '@typescript-eslint/no-floating-promises': 'error',
    },
  },
]);
