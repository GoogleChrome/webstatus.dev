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

import {LitElement, html, type TemplateResult, CSSResultGroup, css} from 'lit';
import {customElement} from 'lit/decorators.js';
import {SHARED_STYLES} from '../css/shared-css.js';

@customElement('webstatus-not-found-error-page')
export class WebstatusNotFoundErrorPage extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        #error-container {
          width: 100%;
          height: 100%;
          flex-direction: column;
          justify-content: center;
          align-items: center;
          gap: 48px;
          display: inline-flex;
        }
        #error-header {
          align-self: stretch;
          height: 108px;
          flex-direction: column;
          justify-content: flex-start;
          align-items: center;
          gap: 12px;
          display: flex;
        }
        #error-status-code {
          color: #2563eb;
          font-size: 15px;
          font-weight: 700;
          line-height: 22.5px;
          word-wrap: break-word;
        }
        #error-headline {
          color: #1d2430;
          font-size: 32px;
          font-weight: 700;
          word-wrap: break-word;
        }
        #error-detailed-message {
          color: #6c7381;
          font-size: 15px;
          font-weight: 400;
          line-height: 22.5px;
          word-wrap: break-word;
        }
        #error-actions {
          justify-content: center;
          align-items: center;
          gap: 24px;
          display: inline-flex;
        }
        #error-action-home {
          width: 136px;
          padding-left: 16px;
          padding-right: 16px;
          justify-content: center;
          align-items: center;
          gap: 8px;
          display: flex;
        }
        #error-action-report {
          width: 145px;
          padding-left: 16px;
          padding-right: 16px;
          border-radius: 4px;
          justify-content: center;
          align-items: center;
          gap: 8px;
          display: flex;
        }

        #error-action-report a {
          color: inherit;
          text-decoration: none;
        }
      `,
    ];
  }
  protected render(): TemplateResult {
    return html`
      <div id="error-container">
        <div id="error-header">
          <div id="error-status-code">404</div>
          <div id="error-headline">Page not found</div>
          <div id="error-detailed-message">
            We couldn't find the page you're looking for.
          </div>
        </div>

        <div id="error-actions">
          <div id="error-action-home">
            <sl-button id="error-action-home-btn" variant="primary" href="/"
              >Go back home</sl-button
            >
          </div>
          <div id="error-action-report">
            <sl-icon name="github"></sl-icon>
            <a
              href="https://github.com/GoogleChrome/webstatus.dev/issues/new"
              target="_blank"
              >Report an issue</a
            >
          </div>
        </div>
      </div>
    `;
  }
}
