/**
 * Copyright 2025 Google LLC
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

import {customElement} from 'lit/decorators.js';
import {ServiceElement} from './service-element.js';
import {provide} from '@lit/context';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../contexts/app-bookmark-info-context.js';
import {Bookmark, DEFAULT_BOOKMARKS} from '../utils/constants.js';
import {getSearchQuery} from '../utils/urls.js';
import {AppLocation, getCurrentLocation} from '../utils/app-router.js';

interface GetLocationFunction {
  (): AppLocation;
}

@customElement('webstatus-bookmarks-service')
export class WebstatusBookmarksService extends ServiceElement {
  @provide({context: appBookmarkInfoContext})
  appBookmarkInfo: AppBookmarkInfo = {};

  _globalBookmarks: Bookmark[];
  _currentGlobalBookmark?: Bookmark;
  currentLocation?: AppLocation;

  // Helper for testing.
  getLocation: GetLocationFunction = getCurrentLocation;

  constructor() {
    super();
    this.currentLocation = this.getLocation();
    this._globalBookmarks = DEFAULT_BOOKMARKS;
    this._currentGlobalBookmark = this.findCurrentBookmarkByQuery(
      this._globalBookmarks,
    );

    window.addEventListener('popstate', this.handlePopState.bind(this));
  }

  private handlePopState() {
    const location = this.getLocation();
    this.currentLocation = location;
    this._currentGlobalBookmark = this.findCurrentBookmarkByQuery(
      this.appBookmarkInfo.globalBookmarks,
    );
    this.refreshAppBookmarkInfo();
  }

  connectedCallback(): void {
    super.connectedCallback();
    this.refreshAppBookmarkInfo();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
  }

  findCurrentBookmarkByQuery(bookmarks?: Bookmark[]): Bookmark | undefined {
    const currentQuery = getSearchQuery(this.currentLocation ?? {search: ''});
    return bookmarks?.find(bookmark => bookmark.query === currentQuery);
  }

  // Assign the appBookmarkInfo object to trigger a refresh of subscribed contexts
  refreshAppBookmarkInfo() {
    this.appBookmarkInfo = {
      globalBookmarks: this._globalBookmarks,
      currentGlobalBookmark: this._currentGlobalBookmark,
    };
  }
}
