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

import {css} from 'lit';

export const THEME = css`
  :host {
    --color-primary: var(--sl-color-primary-600);
    --color-strong-primary: var(--sl-color-primary-800);
    --color-strongest: var(--sl-color-gray-900);
    --color-strong: var(--sl-color-gray-600);
    --color-medium: var(--sl-color-gray-400);
    --color-light: var(--sl-color-gray-200);
    --color-lightest: var(--sl-color-gray-50);
    --color-background: white;
    --color-highlight-1: var(--sl-color-gray-100);
    --color-highlight-2: var(--sl-color-gray-300);

    --color-gray-bg: var(--sl-color-gray-100);
    --color-gray-fg: var(--sl-color-gray-600);
    --color-green-bg: var(--sl-color-green-100);
    --color-green-fg: var(--sl-color-green-600);
    --color-blue-bg: var(--sl-color-blue-100);
    --color-blue-fg: var(--sl-color-blue-600);
    --color-red-bg: var(--sl-color-red-100);
    --color-red-fg: var(--sl-color-red-600);
  }

  :host {
    --default-color: var(--color-strongest);
    --unimportant-text-color: var(--color-strong);

    --content-padding-large: 24px;
    --content-padding: 16px;
    --content-padding-half: 8px;
    --content-padding-quarter: 4px;

    --border-radius: 8px;
    --logo-color: var(--default-color);
    --logo-size: 32px;
    --icon-size: 22px;
    --link-color: var(--color-strong-primary);
    --link-hover-color: var(--link-color);

    --default-border: 1px solid var(--color-light);

    --card-background: white;
    --card-border-width: 1px;
    --card-border-color: var(--color-light);
    --card-radius: var(--border-radius);

    --table-header-background: var(--color-lightest);
    --table-row-background: var(--color-background);
    --table-divider: var(--default-border);
    --table-border: var(--default-border);
    --table-radius: var(--border-radius);

    --pagination-active-background: var(--color-lightest);

    --chip-border: none;
    --chip-radius: 9999px;
    --chip-background-limited: var(--color-gray-bg);
    --chip-color-limited: var(--color-gray-fg);
    --chip-background-newly: var(--color-blue-bg);
    --chip-color-newly: var(--color-blue-fg);
    --chip-background-widely: var(--color-green-bg);
    --chip-color-widely: var(--color-green-fg);

    --chip-small-font-size: 60%;
    --chip-background-unchanged: var(--color-gray-bg);
    --chip-color-unchanged: var(--color-gray-fg);
    --chip-background-increased: var(--color-green-bg);
    --chip-color-incresed: var(--color-green-fg);
    --chip-background-decreased: var(--color-red-bg);
    --chip-color-decresed: var(--color-red-fg);
  }
`;
