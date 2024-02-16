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

import {LitElement, type TemplateResult, html, CSSResultGroup, css} from 'lit';
import {customElement} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import './webstatus-sidebar.js';

@customElement('webstatus-page')
export class WebstatusPage extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        @media (max-width: 768px) {
          webstatus-sidebar {
            display: none;
          }
        }
        .container {
          display: flex;
          flex-direction: row;
          height: 100%;
          width: 100%;
        }

        .page-container {
          flex: 2;
          padding-left: 20px;
          padding-right: 20px;
          padding-top: 10px;
        }

        webstatus-sidebar {
          flex: 1;
          align-self: stretch;
          max-width: 288px;
          padding-right: 20px;
          padding-top: 10px;
        }

        @media (max-width: 768px) {
          .container {
            flex-direction: column;
          }
        }
      `,
    ];
  }
  protected render(): TemplateResult {
    return html` <div class="container">
      <webstatus-sidebar></webstatus-sidebar>
      <div class="page-container"><slot></slot></div>
    </div>`;
  }
}
