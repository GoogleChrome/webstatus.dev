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
import {customElement, state} from 'lit/decorators.js';
import {SlTree, SlTreeItem} from '@shoelace-style/shoelace';
import {formatOverviewPageUrl} from '../utils/urls.js';
import {
  AppLocation,
  getCurrentLocation,
  navigateToUrl,
} from '../utils/app-router.js';

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

export interface Bookmark {
  // Display name
  name: string;
  // Query for filtering
  query: string;
}

interface GetLocationFunction {
  (): AppLocation;
}

interface NavigateToUrlFunction {
  (url: string): void;
}

const defaultBookmarks: Bookmark[] = [
  {name: 'Baseline 2023', query: 'baseline_date:2023-01-01..2023-12-31'},
];

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

  getLocation: GetLocationFunction = getCurrentLocation;
  navigate: NavigateToUrlFunction = navigateToUrl;

  connectedCallback(): void {
    super.connectedCallback();
    this.downloadBookmarks();
    this.updateActiveStatus();
  }

  @state()
  bookmarks: Bookmark[] = [];

  @state()
  private activeBookmarkQuery: string | null = null;

  updateActiveStatus(): void {
    const location = this.getLocation();
    const queryParams = new URLSearchParams(location.search);
    const currentQuery = queryParams.get('q');

    this.activeBookmarkQuery =
      this.bookmarks.find(bookmark => bookmark.query === currentQuery)?.query ||
      null;
  }

  getActiveBookmarkQuery(): string | null {
    return this.activeBookmarkQuery;
  }

  downloadBookmarks() {
    // If we did not set any bookmarks, "download" (future) and add the default bookmarks.
    if (this.bookmarks.length === 0) {
      // In the future, we can get more bookmarks from the backend and combine with the default list here.
      this.setBookmarks(defaultBookmarks);
    }
  }

  setBookmarks(newBookmarks: Bookmark[]) {
    this.bookmarks = newBookmarks;
  }

  private handleBookmarkClick(bookmark: Bookmark) {
    const newUrl = formatOverviewPageUrl(this.getLocation(), {
      q: bookmark.query,
    });

    this.navigate(newUrl);

    this.updateActiveStatus();
  }

  firstUpdated(): void {
    const tree = this.shadowRoot!.querySelector('sl-tree') as SlTree;
    if (!tree) {
      throw new Error('No tree found');
    }

    // Reselect the sl-tree-item corresponding to the current URL path.
    const currentUrl = new URL(this.getLocation().href);
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

      if (currentUrl.href !== this.getLocation().href) {
        this.navigate(currentUrl.href);
      }
    });
  }

  renderBookmark(bookmark: Bookmark, index: number): TemplateResult {
    const isQueryActive = this.activeBookmarkQuery === bookmark.query;
    const bookmarkIcon = isQueryActive ? 'bookmark-star' : 'bookmark';
    const bookmarkId = `bookmark${index}`;
    return html`
      <sl-tree-item
        id=${bookmarkId}
        ?selected=${isQueryActive}
        @click=${() => this.handleBookmarkClick(bookmark)}
      >
        <sl-icon name="${bookmarkIcon}"></sl-icon> ${bookmark.name}
      </sl-tree-item>
    `;
  }

  render(): TemplateResult {
    return html`
      <sl-tree>
        <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
        <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>

        <sl-tree-item id="${NavigationItemKey.FEATURES}">
          <sl-icon name="menu-button"></sl-icon> Features
          ${this.bookmarks.map((bookmark, index) =>
            this.renderBookmark(bookmark, index)
          )}
        </sl-tree-item>
        <sl-tree-item id="${NavigationItemKey.STATISTICS}">
          <sl-icon name="heart-pulse"></sl-icon> Statistics
        </sl-tree-item>
      </sl-tree>
    `;
  }
}
