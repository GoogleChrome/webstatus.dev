/**
 * Copyright 2024 Google LLC
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
import globals from 'globals';
import {configs as liteslint} from 'eslint-plugin-lit';
import gts from 'gts';

export default defineConfig([
  tseslint.configs.recommended,
  ...gts,
  {
    ignores: [
      '**/rollup.config.mjs',
      'dist/*',
      'scripts/*',
      'build/*',
      '**/web-test-runner.config.mjs',
      'coverage/lcov-report/*',
    ],
  },
  {
    extends: [liteslint['flat/recommended']],
  },
  {
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },

    rules: {
      indent: 'off',
      '@typescript-eslint/indent': 'off',
      '@typescript-eslint/space-before-function-paren': 'off',
      'node/no-unpublished-import': ['off'],

      // Strict Type Safety
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unsafe-type-assertion': 'error',
      '@typescript-eslint/no-unnecessary-type-assertion': 'error',

      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
        },
      ],

      eqeqeq: ['error', 'allow-null'],

      'n/no-unpublished-import': [
        'error',
        {
          allowModules: [
            '@open-wc/testing',
            'sinon',
            'openapi-typescript-helpers',
          ],
        },
      ],

      // For CustomEvent. Remove once we upgrade to a LTS version of Node >= 22.1.0.
      'n/no-unsupported-features/node-builtins': 'off',
      '@typescript-eslint/no-restricted-types': [
        'error',
        {
          types: {
            unknown:
              'Prefer explicit types or type guards instead of unknown. Use unknown only in tests or where absolutely unavoidable.',
          },
        },
      ],
    },
  },
  {
    // chai assertion statements will trigger this.
    files: ['**/test/*.test.ts'],
    rules: {
      '@typescript-eslint/no-unused-expressions': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unsafe-type-assertion': 'off',
      '@typescript-eslint/no-unnecessary-type-assertion': 'off',
      '@typescript-eslint/no-restricted-types': 'off',
    },
  },
]);
