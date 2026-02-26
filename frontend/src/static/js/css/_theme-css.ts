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
    /* Base Variables - Using semantic Shoelace variables that auto-toggle globally */
    --color-primary: var(--sl-color-primary-600);
    --color-strong-primary: var(--sl-color-primary-800);
    --color-strongest: var(--sl-color-neutral-900);
    --color-strong: var(--sl-color-neutral-600);
    --color-medium: var(--sl-color-neutral-400);
    --color-light: var(--sl-color-neutral-200);
    --color-lightest: var(--sl-color-neutral-50);
    --color-background: var(--sl-color-neutral-0);
    --color-highlight-1: var(--sl-color-neutral-100);
    --color-highlight-2: var(--sl-color-neutral-300);

    --color-gray-bg: var(--sl-color-neutral-100);
    --color-gray-fg: var(--sl-color-neutral-600);
    --color-green-bg: var(--sl-color-green-100);
    --color-green-fg: var(--sl-color-green-600);
    --color-blue-bg: var(--sl-color-blue-100);
    --color-blue-fg: var(--sl-color-blue-600);
    --color-red-bg: var(--sl-color-red-100);
    --color-red-fg: var(--sl-color-red-600);
    --color-yellow-fg: var(--sl-color-yellow-800);
    --border-color: var(--color-light);
    --header-background: var(--color-background);
    --heading-color: var(--color-strongest);
    --heading-background: transparent;

    --default-color: var(--color-strongest);
    --unimportant-text-color: var(--sl-color-neutral-700);

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
    --browser-logo-size: 24px;
    --platform-logo-size: 18px;

    --default-border-color: var(--border-color);
    --default-border: 1px solid var(--default-border-color);

    --card-background: var(--sl-color-neutral-0);
    --card-border-width: 1px;
    --card-border-color: var(--color-light);
    --card-radius: var(--border-radius);

    --table-header-background: var(--color-lightest);
    --table-header-hover-background: var(--color-blue-bg);
    --table-row-background: var(--color-background);
    --table-divider: var(--default-border);
    --table-border: var(--default-border);
    --table-radius: var(--border-radius);

    --pagination-active-background: var(--color-light);
    --pagination-hover-background: var(--color-blue-bg);

    --chip-border: none;
    --chip-radius: 9999px;
    --chip-background-limited: var(--color-gray-bg);
    --chip-color-limited: var(--color-gray-fg);
    --chip-background-newly: var(--color-blue-bg);
    --chip-color-newly: var(--color-blue-fg);
    --chip-background-widely: var(--color-green-bg);
    --chip-color-widely: var(--color-green-fg);

    --icon-color-avail-unavailable: var(--color-red-fg);

    --chip-small-font-size: 60%;
    --chip-background-unchanged: var(--color-gray-bg);
    --chip-color-unchanged: var(--color-gray-fg);
    --chip-background-increased: var(--color-green-bg);
    --chip-color-increased: var(--color-green-fg);
    --chip-background-decreased: var(--color-red-bg);
    --chip-color-decreased: var(--color-red-fg);
  }

  /* Specific Overrides for Dark Mode */
  :host-context(.sl-theme-dark),
  :host(.sl-theme-dark),
  .sl-theme-dark {
    --color-primary: var(--sl-color-primary-500);
    --color-strong-primary: var(--sl-color-primary-400);

    /* Dark mode variants for color chips */
    --color-green-bg: var(--sl-color-green-900);
    --color-green-fg: var(--sl-color-green-400);
    --color-blue-bg: var(--sl-color-blue-900);
    --color-blue-fg: var(--sl-color-blue-400);
    --color-red-bg: var(--sl-color-red-900);
    --color-red-fg: var(--sl-color-red-400);
    --color-yellow-fg: var(--sl-color-yellow-400);

    --header-background: transparent;
    --heading-color: white;
    --default-color: white;

    --link-color: var(--sl-color-primary-800);
    --link-hover-color: var(--sl-color-primary-700);
  }
`;
