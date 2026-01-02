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
  nothing,
} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {SlTree, SlTreeItem} from '@shoelace-style/shoelace';
import {
  formatOverviewPageUrl,
  getSearchID,
  getSearchQuery,
} from '../utils/urls.js';
import {
  AppLocation,
  getCurrentLocation,
  navigateToUrl,
} from '../utils/app-router.js';

import {
  GITHUB_REPO_ISSUE_LINK,
  ABOUT_PAGE_LINK,
  BookmarkOwnerRole,
  GlobalSavedSearch,
  UserSavedSearch,
} from '../utils/constants.js';
import {consume} from '@lit/context';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
  savedSearchHelpers,
} from '../contexts/app-bookmark-info-context.js';
import {TaskStatus} from '@lit/task';
import {User} from 'firebase/auth';
import {firebaseUserContext} from '../contexts/firebase-user-context.js';

// Map from sl-tree-item ids to paths.
enum NavigationItemKey {
  FEATURES = 'features-item',
  STATISTICS = 'statistics-item',
  NOTIFICATION_CHANNELS = 'notification-channels-item',
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
  [NavigationItemKey.NOTIFICATION_CHANNELS]: {
    id: NavigationItemKey.NOTIFICATION_CHANNELS,
    path: '/settings/notification-channels',
  },
};

interface GetLocationFunction {
  (): AppLocation;
}

interface NavigateToUrlFunction {
  (url: string, event?: MouseEvent): void;
}

type savedSearchIconType = 'bookmark' | 'bookmark-star';

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
        .saved-search-link {
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
        sl-skeleton {
          width: 10rem;
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
  activeQuery: string | null = null;

  @consume({context: appBookmarkInfoContext, subscribe: true})
  @state()
  appBookmarkInfo?: AppBookmarkInfo;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user: User | null | undefined;

  // For now, unconditionally open the features dropdown.
  @state()
  private isFeaturesDropdownExpanded: boolean = true;

  updateActiveStatus(): void {
    this.highlightNavigationItem(this.getNavTree());
    // Check if activeQuery needs to be updated
    const newActiveQuery =
      savedSearchHelpers.getCurrentQuery(this.appBookmarkInfo) || null;

    this.activeQuery = newActiveQuery;
    this.requestUpdate();
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
      // Clear out any search parameters that may have been set by the saved searches.
      currentUrl.search = '';

      if (currentUrl.href !== this.getLocation().href) {
        this.navigate(currentUrl.href);
      }
    });
  }

  getSavedSearchID(index: number, type: string): string {
    return `${type}bookmark${index}`;
  }
  getSavedSearchIcon(isQueryActive: boolean): savedSearchIconType {
    return isQueryActive ? 'bookmark-star' : 'bookmark';
  }

  renderGlobalSavedSearch(
    savedSearch: GlobalSavedSearch,
    index: number,
  ): TemplateResult {
    const savedSearchId = this.getSavedSearchID(index, 'global');
    const currentLocation = this.getLocation();
    const currentURL = new URL(currentLocation.href);

    let savedSearchUrl;
    if (savedSearch.override_num_param) {
      savedSearchUrl = formatOverviewPageUrl(currentURL, {
        q: savedSearch.query,
        start: 0,
        num: savedSearch.override_num_param,
        // If the user is on a saved search and clicks on a global saved search,
        // we should clear the search id parameter.
        search_id: '',
      });
    } else {
      savedSearchUrl = formatOverviewPageUrl(currentURL, {
        q: savedSearch.query,
        start: 0,
        // If the user is on a saved search and clicks on a global saved search,
        // we should clear the search id parameter.
        search_id: '',
      });
    }
    // The saved search should only be active when the path is the FEATURES path
    // and the query is set to the active query.
    const isQueryActive =
      currentURL.pathname === navigationMap[NavigationItemKey.FEATURES].path &&
      getSearchQuery(currentLocation) === this.activeQuery &&
      savedSearch.query === this.activeQuery;
    const icon = this.getSavedSearchIcon(isQueryActive);

    return html`
      <sl-tree-item id=${savedSearchId} ?selected=${isQueryActive}>
        <a class="saved-search-link" href="${savedSearchUrl}">
          <sl-icon name="${icon}"></sl-icon> ${savedSearch.name}
        </a>
      </sl-tree-item>
    `;
  }

  renderUserSavedSearch(
    savedSearch: UserSavedSearch,
    index: number,
  ): TemplateResult {
    const savedSearchId = this.getSavedSearchID(index, 'user');
    const currentLocation = this.getLocation();
    const currentURL = new URL(currentLocation.href);
    let savedSearchEditUrl;
    const savedSearchUrl = formatOverviewPageUrl(currentURL, {
      start: 0,
      search_id: savedSearch.id,
      // If the user is on a saved search and clicks on a global saved search,
      // we should clear the q parameter.
      q: '',
    });
    if (savedSearch.permissions?.role === BookmarkOwnerRole) {
      savedSearchEditUrl = formatOverviewPageUrl(currentURL, {
        start: 0,
        search_id: savedSearch.id,
        edit_saved_search: true,
        // If the user is on a saved search and clicks on a global saved search,
        // we should clear the q parameter.
        q: '',
      });
    }

    // The savedSearch should only be active when the path is the FEATURES path,
    // the search ID is set to the active search ID and the query is set to the active query.
    const isQueryActive =
      currentURL.pathname === navigationMap[NavigationItemKey.FEATURES].path &&
      (getSearchQuery(currentLocation) === this.activeQuery ||
        savedSearch.id === getSearchID(currentLocation)) &&
      savedSearch.query === this.activeQuery;
    const icon = this.getSavedSearchIcon(isQueryActive);

    return html`
      <sl-tree-item id=${savedSearchId} ?selected=${isQueryActive}>
        <a class="saved-search-link" href="${savedSearchUrl}">
          <sl-icon name="${icon}"></sl-icon> ${savedSearch.name}
        </a>
        ${savedSearchEditUrl
          ? html` <sl-icon-button
              name="pencil"
              label="Edit"
              class="saved-search-edit-link"
              href="${savedSearchEditUrl}"
            ></sl-icon-button>`
          : nothing}
      </sl-tree-item>
    `;
  }

  renderUserSavedSearches(): TemplateResult {
    if (this.appBookmarkInfo?.userSavedSearchesTask === undefined) {
      return html``;
    }
    // If there is no data, render nothing
    if (
      this.appBookmarkInfo.userSavedSearchesTask.status ===
        TaskStatus.COMPLETE &&
      !this.appBookmarkInfo?.userSavedSearchesTask.data
    ) {
      return html`${nothing}`;
    }
    let section: TemplateResult = html``;
    if (
      this.appBookmarkInfo?.userSavedSearchesTask.status ===
        TaskStatus.INITIAL ||
      this.appBookmarkInfo?.userSavedSearchesTask.status === TaskStatus.PENDING
    ) {
      section = html`
        <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
        <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
        <sl-tree-item><sl-skeleton effect="sheen"></sl-skeleton></sl-tree-item>
      `;
    }
    if (
      this.appBookmarkInfo.userSavedSearchesTask.status ===
        TaskStatus.COMPLETE &&
      this.appBookmarkInfo?.userSavedSearchesTask.data
    ) {
      section = html` ${this.appBookmarkInfo?.userSavedSearchesTask.data?.map(
        (savedSearch, index) => this.renderUserSavedSearch(savedSearch, index),
      )}`;
    }
    return html`
      <sl-divider aria-hidden="true"></sl-divider>
      <sl-tree-item id="your-bookmarks-list" .expanded=${true}>
        Your Bookmarks ${section}
      </sl-tree-item>
    `;
  }

  renderSettingsMenu(): TemplateResult {
    if (this.user === undefined) {
      return html`${nothing}`;
    }
    if (this.user === null) {
      return html`${nothing}`;
    }

    return html`
      <sl-divider></sl-divider>
      <sl-tree-item id="notifications-channels-item">
        <sl-icon name="mailbox-flag"></sl-icon>
        <a
          class="features-link"
          href="${navigationMap[NavigationItemKey.NOTIFICATION_CHANNELS].path}"
        >
          Notification Channels
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
          ${this.appBookmarkInfo?.globalSavedSearches?.map(
            (savedSearch, index) =>
              this.renderGlobalSavedSearch(savedSearch, index),
          )}
        </sl-tree-item>
        <!-- commented out rather than merely hidden, to avoid breaking sl-tree
        <sl-tree-item id="{NavigationItemKey.STATISTICS}">
          <sl-icon name="heart-pulse"></sl-icon> Statistics
        </sl-tree-item> -->
        ${this.renderUserSavedSearches()} ${this.renderSettingsMenu()}

        <sl-divider aria-hidden="true"></sl-divider>
        <sl-tree-item class="report-issue-item">
          <sl-icon name="graph-up"></sl-icon>
          <a class="report-issue-link" href="/stats">Platform statistics</a>
        </sl-tree-item>
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
