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
  html
} from 'lit'
import { customElement } from 'lit/decorators.js'

import { SHARED_STYLES } from '../css/shared-css.js'
import { DRAWER_WIDTH_PX, IS_MOBILE } from './utils.js'
import './webstatus-login.js'



@customElement('webstatus-header')
export class WebstatusHeader extends LitElement {
  static get styles(): CSSResultGroup {
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
      `
    ]
  }

  _fireEvent(eventName: string, detail: CustomEventInit | undefined): void {
    console.info(`Firing event: ${eventName}`)
    const event = new CustomEvent(eventName, {
      bubbles: true,
      composed: true,
      detail
    })
    this.dispatchEvent(event)
  }

  handleDrawer(): void {
    this._fireEvent('drawer-clicked', {})
  }

  render(): TemplateResult {
    return html`
      <header>
        <div class="title">
          ${this.renderHamburger()}
          <img
            class="website-logo"
            src="https://fakeimg.pl/400x400?text=LOGO"
          />
          <div class="website-title">Web Platform Dashboard</div>
        </div>

        ${this.renderDrawer()}

        <nav class="nav-links">
          <a href="/">Features</a>
          <a href="#">About</a>
          <a href="#">Stats</a>
        </nav>

        <div class="sign-in">
          <webstatus-login></webstatus-login>
        </div>
      </header>
    `
  }

  renderDrawer(): TemplateResult {
    if (IS_MOBILE) {
      return html`
        <sl-drawer
          label="Menu"
          placement="start"
          class="drawer-placement-start"
          style="--size: ${DRAWER_WIDTH_PX}px;"
          contained
          noHeader
          @drawer-clicked="${this.toggleDrawer}"
        >
          >
          <webstatus-overview-sidebar></webstatus-overview-sidebar>
        </sl-drawer>
      `
    } else {
      return html``
    }
  }

  renderHamburger(): TemplateResult {
    if (IS_MOBILE) {
      return html`
        <sl-icon-button
          data-testid="menu"
          variant="text"
          class="menu"
          style="font-size: 2.4rem;"
          @click="${this.handleDrawer}"
          name="list"
        >
        </sl-icon-button>
      `
    } else {
      return html``
    }
  }

  toggleDrawer(): void {
    const drawer = this.shadowRoot?.querySelector('sl-drawer')
    if (drawer?.open === true) {
      void drawer.hide()
    } else {
      if (drawer != null) void drawer.show()
    }
  }
}
