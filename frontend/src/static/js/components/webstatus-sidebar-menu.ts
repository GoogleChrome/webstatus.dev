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
import {SlTree, SlTreeItem} from '@shoelace-style/shoelace';

// Map from sl-tree-item ids to paths.
enum NavigationItemKey {
  FEATURES = 'features-item',
  STATISTICS = 'statistics-item',
}

interface NavigationItem {
  id: string;
  path: string;
}

type NavigationMap = {
  [key in NavigationItemKey]: NavigationItem;
};

const navigationMap: NavigationMap = {
  [NavigationItemKey.FEATURES]: {
    id: NavigationItemKey.FEATURES,
    path: '/',
  },
  [NavigationItemKey.STATISTICS]: {
    id: NavigationItemKey.STATISTICS,
    path: '/stats',
  },
};

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
    const tree = this.shadowRoot!.querySelector('sl-tree') as SlTree;
    if (!tree) {
      throw new Error('No tree found');
    }

    // Reselect the sl-tree-item corresponding to the current URL path.
    const currentUrl = new URL(window.location.href);
    const currentPath = currentUrl.pathname;
    const matchingNavItem = Object.values(navigationMap).find(
      item => item.path === currentPath
    );

    if (matchingNavItem) {
      const itemToSelect = tree.querySelector(
        `#${matchingNavItem.id}`
      ) as SlTreeItem;
      if (itemToSelect) {
        itemToSelect.selected = true;
      }
    }

    tree!.addEventListener('sl-selection-change', () => {
      const selectedItems = tree.selectedItems;
      if (selectedItems.length <= 0) {
        return;
      }
      const selectedItem = selectedItems[0];
      const navigationItem =
        navigationMap[selectedItem.id as NavigationItemKey];
      if (!navigationItem) {
        return;
      }
      currentUrl.pathname = navigationItem.path;

      if (currentUrl.href !== window.location.href) {
        window.location.href = currentUrl.href;
      }
    });
  }

  render(): TemplateResult {
    return html`
      <sl-tree>
        <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
        <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>

        <sl-tree-item id="${NavigationItemKey.FEATURES}">
          <sl-icon name="menu-button"></sl-icon> Features
          <sl-tree-item>
            <sl-icon name="bookmark"></sl-icon> Baseline 2023
          </sl-tree-item>
        </sl-tree-item>
        <sl-tree-item id="${NavigationItemKey.STATISTICS}">
          <sl-icon name="heart-pulse"></sl-icon> Statistics
        </sl-tree-item>
      </sl-tree>
    `;
  }
}
