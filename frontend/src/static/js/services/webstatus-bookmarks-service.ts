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

import {customElement, state} from 'lit/decorators.js';
import {ServiceElement} from './service-element.js';
import {consume, provide} from '@lit/context';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../contexts/app-bookmark-info-context.js';
import {Bookmark, DEFAULT_BOOKMARKS} from '../utils/constants.js';
import {getSearchID, getSearchQuery} from '../utils/urls.js';
import {AppLocation, getCurrentLocation} from '../utils/app-router.js';
import {Task, TaskStatus} from '@lit/task';
import {User} from 'firebase/auth';
import {APIClient} from '../api/client.js';
import {apiClientContext} from '../contexts/api-client-context.js';
import {firebaseUserContext} from '../contexts/firebase-user-context.js';
import {ApiError} from '../api/errors.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {PropertyValueMap} from 'lit';

interface GetLocationFunction {
  (): AppLocation;
}

@customElement('webstatus-bookmarks-service')
export class WebstatusBookmarksService extends ServiceElement {
  @provide({context: appBookmarkInfoContext})
  appBookmarkInfo: AppBookmarkInfo = {};

  @consume({context: apiClientContext, subscribe: true})
  @state()
  apiClient?: APIClient;
  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user?: User;

  _globalBookmarks: Bookmark[];
  _currentGlobalBookmark?: Bookmark;
  _currentUserSavedBookmark?: Bookmark;
  _userSavedBookmarkByIDTaskTracker?: TaskTracker<
    Bookmark,
    Error & {message: string}
  > = undefined;

  loadingUserSavedBookmarkByIDTask = new Task(this, {
    args: () => [this.currentLocation, this.apiClient, this.user] as const,
    task: async ([location, apiClient, user]) => {
      const searchID = getSearchID(location ?? {search: ''});
      if (!searchID || !apiClient) {
        return;
      }
      let token: string | undefined;
      if (user) {
        token = await user.getIdToken();
      }
      this._userSavedBookmarkByIDTaskTracker = {
        status: TaskStatus.PENDING,
        data: undefined,
        error: undefined,
      };
      this.refreshAppBookmarkInfo();

      return await apiClient.getSavedSearchByID(searchID, token);
    },
    onComplete: data => {
      this._userSavedBookmarkByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: data,
        error: undefined,
      };
      this.refreshAppBookmarkInfo();
    },
    onError: async (error: unknown) => {
      let msg: string;
      if (error instanceof ApiError) {
        msg = error.message;
      } else {
        msg = 'Unknown message. Check console for details.';
        console.error(error);
      }

      const searchID = getSearchID(location ?? {search: ''});
      this._userSavedBookmarkByIDTaskTracker = {
        status: TaskStatus.ERROR,
        error: new Error(`Error fetching saved search ID ${searchID}: ${msg}`),
        data: undefined,
      };
      this.refreshAppBookmarkInfo();
    },
  });

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
      userSavedSearchBookmarkTask: this._userSavedBookmarkByIDTaskTracker,
    };
  }

  protected willUpdate(_changedProperties: PropertyValueMap<this>) {
    console.log(_changedProperties);
  }
}
