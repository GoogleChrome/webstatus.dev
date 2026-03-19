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
import {DRAWER_WIDTH_PX, IS_MOBILE} from './utils.js';
import './webstatus-sidebar.js';

@customElement('webstatus-page')
export class WebstatusPage extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        :host {
          flex-shrink: 1;
          min-width: 0;
          box-sizing: border-box;
          width: 100%;
          overflow: hidden;
        }
        .container {
          height: 100%;
          width: 100%;
          overflow: hidden;
        }

        #sidebar-drawer {
          max-width: ${DRAWER_WIDTH_PX}px;
          border-right: var(--default-border);
          flex-shrink: 0;
        }
        #sidebar-drawer:not([open]),
        #sidebar-drawer[no-header]::part(header) {
          display: none;
        }

        webstatus-sidebar {
          padding-right: 20px;
          padding-top: 10px;
        }

        .page-container {
          padding: var(--content-padding);
          flex-grow: 1;
          flex-shrink: 1;
          min-width: 0;
          overflow-x: hidden;
          box-sizing: border-box;
        }
      `,
    ];
  }

  firstUpdated(): void {
    const sidebarDrawer = this.shadowRoot?.querySelector('#sidebar-drawer');
    if (!(sidebarDrawer instanceof HTMLElement)) {
      throw new Error('Sidebar Drawer is missing');
    }

    const showSidebarDrawer = () => {
      if ('show' in sidebarDrawer && typeof sidebarDrawer.show === 'function') {
        void sidebarDrawer.show();
      } else {
        sidebarDrawer.setAttribute('open', '');
      }
    };

    if (!IS_MOBILE) {
      showSidebarDrawer();
    }

    document.addEventListener('toggle-menu', () => {
      if (sidebarDrawer.hasAttribute('open')) {
        if (
          'hide' in sidebarDrawer &&
          typeof sidebarDrawer.hide === 'function'
        ) {
          void sidebarDrawer.hide();
        } else {
          sidebarDrawer.removeAttribute('open');
        }
      } else {
        showSidebarDrawer();
      }
    });
  }

  protected render(): TemplateResult {
    return html` <div class="container hbox valign-items-top">
      <sl-details
        id="sidebar-drawer"
        class="valign-stretch halign-shrink-0"
        no-header
      >
        <webstatus-sidebar></webstatus-sidebar>
      </sl-details>
      <div class="page-container vbox halign-stretch"><slot></slot></div>
    </div>`;
  }
}
