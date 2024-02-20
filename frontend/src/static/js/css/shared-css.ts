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

import {css} from 'lit';

import {RESET} from './_reset-css.js';
import {THEME} from './_theme-css.js';
import {FLEX_BOX} from './flex-box.js';

export const SHARED_STYLES = [
  RESET,
  THEME,
  FLEX_BOX,
  css`
    :host {
      font-family: ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji',
        'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';
      color: var(--default-color);
    }

    a {
      color: var(--link-color);
      text-decoration: none;
    }
    a:hover {
      color: var(--link-hover-color);
      text-decoration: underline;
    }

    .data-table {
      width: 100%;
      border: var(--default-border);
      border-radius: var(--border-radius);
    }
    .data-table th {
      text-align: left;
      background: var(--table-header-background);
      padding: var(--content-padding-half) var(--content-padding);
    }
    .data-table td {
      border-top: var(--default-border);
      padding: var(--content-padding-half) var(--content-padding);
    }

    .chip {
      border: var(--chip-border);
      border-radius: var(--chip-radius);
      white-space: nowrap;
      padding: var(--content-padding-quarter) var(--content-padding-half);
    }

    h1 {
      font-weight: 700;
      font-size: 32px;
    }

    h2 {
      font-weight: 700;
      font-size: 24px;
    }

    h3,
    h4 {
      font-weight: 300;
    }

    h2,
    h3,
    h4 {
      background: var(--heading-background);
      color: var(--heading-color);
    }

    h3 {
      font-size: 20px;
    }

    a {
      text-decoration: none;
      color: var(--link-color);
    }
    a:hover {
      text-decoration: underline;
      color: var(--link-hover-color);
      cursor: pointer;
    }
  `,
];
