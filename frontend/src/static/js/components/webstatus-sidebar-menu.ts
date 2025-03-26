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
  PropertyValueMap,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SlTree, SlTreeItem} from '@shoelace-style/shoelace';
import {formatOverviewPageUrl} from '../utils/urls.js';
import {
  AppLocation,
  getCurrentLocation,
  navigateToUrl,
} from '../utils/app-router.js';

import {
  GITHUB_REPO_ISSUE_LINK,
  ABOUT_PAGE_LINK,
  Bookmark,
} from '../utils/constants.js';
import {consume} from '@lit/context';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
  bookmarkHelpers,
} from '../contexts/app-bookmark-info-context.js';

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

interface GetLocationFunction {
  (): AppLocation;
}

interface NavigateToUrlFunction {
  (url: string, event?: MouseEvent): void;
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
        .features-link {
          color: inherit;
          text-decoration: none;
        }
        .bookmark-link {
          color: inherit;
          text-decoration: none;
        }

        .report-issue-item {
          margin-top: auto;
        }

        .report-issue-link {
          color: inherit;
          text-decoration: none;
        }
        .about-link {
          color: inherit;
          text-decoration: none;
        }
      `,
    ];
  }

  getLocation: GetLocationFunction = getCurrentLocation;
  navigate: NavigateToUrlFunction = navigateToUrl;

  constructor() {
    super();
  }

  connectedCallback(): void {
    super.connectedCallback();
    this.updateActiveStatus();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
  }

  private handleBookmarkInfoUpdate() {
    this.updateActiveStatus();
  }

  @state()
  private activeBookmarkQuery: string | null = null;

  @consume({context: appBookmarkInfoContext, subscribe: true})
  @state()
  appBookmarkInfo?: AppBookmarkInfo;

  // For now, unconditionally open the features dropdown.
  @state()
  private isFeaturesDropdownExpanded: boolean = true;

  updateActiveStatus(): void {
    this.highlightNavigationItem(this.getNavTree());
    // Check if activeBookmarkQuery needs to be updated
    const newActiveBookmarkQuery =
      bookmarkHelpers.getCurrentBookmark(
        this.appBookmarkInfo,
        this.getLocation(),
      )?.query || null;

    this.activeBookmarkQuery = newActiveBookmarkQuery;
    this.requestUpdate();
  }

  getActiveBookmarkQuery(): string | null {
    return this.activeBookmarkQuery;
  }

  getNavTree(): SlTree | undefined {
    return this.shadowRoot!.querySelector('sl-tree') as SlTree;
  }

  private highlightNavigationItem(tree: SlTree | undefined) {
    if (!tree) {
      return;
    }
    // Reselect the sl-tree-item corresponding to the current URL path.
    const currentUrl = new URL(this.getLocation().href);
    const currentPath = currentUrl.pathname;
    const matchingNavItem = Object.values(navigationMap).find(
      item => item.path === currentPath,
    );

    if (matchingNavItem) {
      const itemToSelect = tree.querySelector(
        `#${matchingNavItem.id}`,
      ) as SlTreeItem;
      if (itemToSelect) {
        itemToSelect.selected = true;
      }
    }
  }

  protected willUpdate(changedProperties: PropertyValueMap<this>): void {
    if (changedProperties.has('appBookmarkInfo')) {
      this.handleBookmarkInfoUpdate();
    }
  }

  firstUpdated(): void {
    const tree = this.getNavTree();
    if (!tree) {
      throw new Error('No tree found');
    }

    this.highlightNavigationItem(tree);

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
      const currentUrl = new URL(this.getLocation().href);
      currentUrl.pathname = navigationItem.path;
      // Clear out any search parameters that may have been set by the bookmarks.
      currentUrl.search = '';

      if (currentUrl.href !== this.getLocation().href) {
        this.navigate(currentUrl.href);
      }
    });
  }

  renderBookmark(bookmark: Bookmark, index: number): TemplateResult {
    const bookmarkId = `bookmark${index}`;
    const currentLocation = this.getLocation();
    const currentURL = new URL(currentLocation.href);

    let bookmarkUrl;
    if (bookmark.override_num_param) {
      bookmarkUrl = formatOverviewPageUrl(currentURL, {
        q: bookmark.query,
        start: 0,
        num: bookmark.override_num_param,
        // If the user is on a saved search and clicks on a global bookmark,
        // we should clear the search id parameter.
        search_id: '',
      });
    } else {
      bookmarkUrl = formatOverviewPageUrl(currentURL, {
        q: bookmark.query,
        start: 0,
        // If the user is on a saved search and clicks on a global bookmark,
        // we should clear the search id parameter.
        search_id: '',
      });
    }
    // The bookmark should only be active when the path is the FEATURES path
    // and the query is set to the active query.
    const isQueryActive =
      currentURL.pathname === navigationMap[NavigationItemKey.FEATURES].path &&
      new URLSearchParams(currentLocation.search).get('q') ===
        this.activeBookmarkQuery &&
      bookmark.query === this.activeBookmarkQuery;
    const bookmarkIcon = isQueryActive ? 'bookmark-star' : 'bookmark';

    return html`
      <sl-tree-item id=${bookmarkId} ?selected=${isQueryActive}>
        <a class="bookmark-link" href="${bookmarkUrl}">
          <sl-icon name="${bookmarkIcon}"></sl-icon> ${bookmark.name}
        </a>
      </sl-tree-item>
    `;
  }

  render(): TemplateResult {
    return html`
      <sl-tree>
        <sl-icon name="caret-right-fill" slot="expand-icon"></sl-icon>
        <sl-icon name="caret-right-fill" slot="collapse-icon"></sl-icon>

        <sl-tree-item
          id="${NavigationItemKey.FEATURES}"
          .expanded=${this.isFeaturesDropdownExpanded}
        >
          <sl-icon name="menu-button"></sl-icon>
          <a
            class="features-link"
            href="${navigationMap[NavigationItemKey.FEATURES].path}"
            >Features</a
          >
          ${this.appBookmarkInfo?.globalBookmarks?.map((bookmark, index) =>
            this.renderBookmark(bookmark, index),
          )}
        </sl-tree-item>
        <!-- commented out rather than merely hidden, to avoid breaking sl-tree
        <sl-tree-item id="{NavigationItemKey.STATISTICS}">
          <sl-icon name="heart-pulse"></sl-icon> Statistics
        </sl-tree-item> -->

        <sl-divider aria-hidden="true"></sl-divider>

        <sl-tree-item class="report-issue-item">
          <sl-icon name="github"></sl-icon>
          <a
            class="report-issue-link"
            href="${GITHUB_REPO_ISSUE_LINK}"
            target="_blank"
            >Report an issue</a
          >
        </sl-tree-item>

        <sl-tree-item class="about-item">
          <sl-icon name="info-circle"></sl-icon>
          <a class="about-link" href="${ABOUT_PAGE_LINK}" target="_blank"
            >About</a
          >
        </sl-tree-item>
      </sl-tree>
    `;
  }
}
