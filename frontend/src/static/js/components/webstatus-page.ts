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
import { SHARED_STYLES } from '../css/shared-css.js';
import { DRAWER_WIDTH_PX, IS_MOBILE } from './utils.js';
import SlDrawer from '@shoelace-style/shoelace/dist/components/drawer/drawer.js';
import './webstatus-sidebar.js';

@customElement('webstatus-page')
export class WebstatusPage extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        .container {
          position: relative; /* for the menu drawer */
          height: 100%;
          width: 100%;
        }

        #sidebar-drawer {
          --size: ${DRAWER_WIDTH_PX}px;
          position: relative;
        }

        #sidebar-drawer::part(base) {
          position: relative;
          width: auto;
        }
        #sidebar-drawer::part(panel) {
          position: relative;
        }

        webstatus-sidebar {
          max-width: 288px;
          padding-right: 20px;
          padding-top: 10px;
        }

        @media (max-width: 768px) {
          /* .container {
            flex-direction: column;
          } */
          webstatus-sidebar.vbox {
            display: none;
          }
        }

        .page-container {
          padding: var(--content-padding);
        }

      `,
    ];
  }


  firstUpdated(): void {
    if (!IS_MOBILE) {
      // Hide the sidebar by default
    }

    document.addEventListener('toggle-menu', () => {
      console.info('got toggle-menu event');
      const sidebarDrawer = this.shadowRoot?.querySelector('#sidebar-drawer') as SlDrawer | null;
      console.info('sidebarDrawer', sidebarDrawer);
      if (!sidebarDrawer) {
        throw new Error('Unable to addEventListener to sidebarDrawer; it is null or undefined.');
      }
      if (sidebarDrawer!.open === true) {
        void sidebarDrawer.hide();
      } else {
        if (sidebarDrawer !== null && sidebarDrawer !== undefined) {
          void sidebarDrawer!.show();
        }
      }
    });
  }

  protected render(): TemplateResult {
    return html` <div class="container hbox valign-items-top">
      <sl-drawer id="sidebar-drawer"
        label="Menu"
        placement="start"
        contained
        no-header
      >
        <webstatus-sidebar></webstatus-sidebar>
      </sl-drawer>
      <div class="page-container vbox halign-stretch"><slot></slot></div>
    </div>`;
  }

}
