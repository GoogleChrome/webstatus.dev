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

import { html, LitElement, css, type TemplateResult, type CSSResultGroup } from 'lit'
import { customElement } from 'lit/decorators.js'
import { SHARED_STYLES } from '../css/shared-css.js'
// import './webstatus-login.js'

@customElement('webstatus-header')
export class WebstatusHeader extends LitElement {
  static override shadowRootOptions = {
    ...LitElement.shadowRootOptions,
    delegatesFocus: true
  }

  static get styles (): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
      header {
        display: flex;
        justify-content: space-between;
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

      nav {
        display: flex;
        align-items: center;
      }

      .nav-links {
        display: flex;
        align-items: center;
      }
      nav a {
        font-size: 16px;
        color: #1f1f1f;
        text-decoration: none;
        margin-left: 20px;
      }

      nav a.active {
        font-weight: bold;
        color: #0b57d0;
        text-decoration: underline;
      }

      .sign-in {
        margin-top: auto;
        margin-bottom: auto;
        margin-right: 1em;
      }
    `]
  }

  render (): TemplateResult {
    return html`
      <header>
        <div class="title">
          <img class="website-logo" src="https://fakeimg.pl/400x400?text=LOGO" />
          <div class="website-title">Web Platform Dashboard</div>
        </div>

        <nav class="nav-links">
          <a href="#">Features</a>
          <a href="#">About</a>
          <a href="#">Stats</a>
        </nav>

        <div class="sign-in">
        </div>
      </header>
    `
  }
}
