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

import typescriptEslint from '@typescript-eslint/eslint-plugin';
import globals from 'globals';
import tsParser from '@typescript-eslint/parser';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import js from '@eslint/js';
import {FlatCompat} from '@eslint/eslintrc';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all,
});

export default [
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
  ...compat.extends('../node_modules/gts/', 'plugin:lit/recommended'),
  {
    plugins: {
      '@typescript-eslint': typescriptEslint,
    },

    languageOptions: {
      globals: {
        ...globals.browser,
      },

      parser: tsParser,
      ecmaVersion: 'latest',
      sourceType: 'module',
    },

    rules: {
      indent: 'off',
      '@typescript-eslint/indent': 'off',
      '@typescript-eslint/space-before-function-paren': 'off',
      'node/no-unpublished-import': ['off'],

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
    },
  },
  {
    // chai assertion statements will trigger this.
    files: ['**/test/*.test.ts'],
    rules: {
      '@typescript-eslint/no-unused-expressions': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
    },
  },
];
