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
} from 'lit';
import {customElement} from 'lit/decorators.js';

// Map from sl-tree-item id to path, and vice versa.
const idPathMap: {[id: string]: string} = {
  'features-item': '/',
  'statistics-item': '/stats',
};
const pathIdMap: {[path: string]: string} = {};
for (const [id, path] of Object.entries(idPathMap)) {
  pathIdMap[path] = id;
}

@customElement('webstatus-sidebar-menu')
export class WebstatusSidebarMenu extends LitElement {
  static get styles(): CSSResultGroup {
    return [
      css`
        .material-icons {
          font-family: 'Material Icons';
          font-size: 24px;
          vertical-align: middle;
        }
      `,
    ];
  }

  firstUpdated(): void {
    const tree = this.shadowRoot?.querySelector('sl-tree');
    if (!tree) {
      throw new Error('No tree found');
    }

    // Reselect the sl-tree-item corresponding to the current URL path.
    const currentPath = new URL(window.location.href).pathname;
    const id = pathIdMap[currentPath];
    const item = tree?.querySelector(`#${id}`);
    if (item) {
      item.setAttribute('selected', '');
    }

    tree!.addEventListener('sl-selection-change', () => {
      const selectedItem = tree!.querySelector('[selected]');
      const id = selectedItem?.id;
      if (id) {
        const path = idPathMap[id];
        if (path) {
          // If the path is different from the current path, update the URL.
          const currentUrl = new URL(window.location.href);
          currentUrl.pathname = path; // Update only the path
          if (currentUrl.href !== window.location.href) {
            window.location.href = currentUrl.href;
          }
        }
      }
    });
  }

  render(): TemplateResult {
    return html`
      <sl-tree>
        <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
        <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>

        <sl-tree-item id="features-item">
          <sl-icon name="menu-button"></sl-icon> Features
          <sl-tree-item>
            <sl-icon name="bookmark"></sl-icon> Baseline 2023
          </sl-tree-item>
          <sl-tree-item>
            <sl-icon name="bookmark"></sl-icon> Top Developer Needs
          </sl-tree-item>
          <sl-tree-item> <sl-icon name="star"></sl-icon> Starred </sl-tree-item>
        </sl-tree-item>
        <sl-tree-item id="statistics-item">
          <sl-icon name="heart-pulse"></sl-icon> Statistics
        </sl-tree-item>
        <sl-tree-item> <sl-icon name="bell"></sl-icon> Updates </sl-tree-item>
        <sl-tree-item>
          <sl-icon name="info-circle"></sl-icon> About
        </sl-tree-item>
      </sl-tree>
    `;
  }
}
