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

import {
  type CSSResultGroup,
  LitElement,
  type TemplateResult,
  css,
  html,
  nothing,
} from 'lit';
import {customElement} from 'lit/decorators.js';

import {SHARED_STYLES} from '../css/shared-css.js';
import './webstatus-login.js';

@customElement('webstatus-header')
export class WebstatusHeader extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        header {
          flex-grow: 1; /* .hbox > .halign-stretch */
          align-items: center;
          background: #f2f2f2;
          height: 94px;
        }
        .title {
          display: flex;
        }

        .website-logo {
          width: 31px;
          height: 31px;
          background-color: #ccc;
          margin-left: 1em;
          margin-top: auto;
          margin-bottom: auto;
        }

        .website-title {
          font-weight: 700;
          font-size: 24px;
          color: #1f1f1f;
          margin-left: 5px;
          margin-top: auto;
          margin-bottom: auto;
        }
      `,
    ];
  }

  _fireEvent(eventName: string, detail: CustomEventInit | undefined): void {
    const event = new CustomEvent(eventName, {
      bubbles: true,
      composed: true,
      detail,
    });
    this.dispatchEvent(event);
  }

  handleDrawer(): void {
    this._fireEvent('toggle-menu', {});
  }

  render(): TemplateResult {
    const signInButton = nothing;
    // <div class="sign-in">
    //   <webstatus-login></webstatus-login>
    // </div>

    return html`
      <header class="hbox">
        <div class="title">
          <sl-icon-button
            id="menu-button"
            data-testid="menu"
            variant="text"
            class="menu"
            style="font-size: 2.4rem;"
            @click="${this.handleDrawer}"
            name="list"
          >
          </sl-icon-button>
          <h2 class="website-title">Web Platform Dashboard</h2>
        </div>

        <div class="spacer"></div>
        ${signInButton}
      </header>
    `;
  }
}
