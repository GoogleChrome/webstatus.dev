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
  SavedSearchError,
  SavedSearchInternalError,
  SavedSearchNotFoundError,
  SavedSearchUnknownError,
  appBookmarkInfoContext,
} from '../contexts/app-bookmark-info-context.js';
import {Bookmark, DEFAULT_BOOKMARKS} from '../utils/constants.js';
import {getSearchID, getSearchQuery, updatePageUrl} from '../utils/urls.js';
import {AppLocation, getCurrentLocation} from '../utils/app-router.js';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {User} from 'firebase/auth';
import {firebaseUserContext} from '../contexts/firebase-user-context.js';
import {Task, TaskStatus} from '@lit/task';
import {NotFoundError, ApiError} from '../api/errors.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {Toast} from '../utils/toast.js';

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

  _userSavedBookmarkByIDTaskTracker?: TaskTracker<Bookmark, SavedSearchError> =
    undefined;

  loadingUserSavedBookmarkByIDTask = new Task(this, {
    args: () => [this._currentLocation, this.apiClient, this.user] as const,
    task: async ([location, apiClient, user]) => {
      const searchID = this.getSearchID(
        location ?? {search: '', pathname: '', href: ''},
      );
      if (!searchID || !apiClient) {
        return;
      }
      if (
        this.appBookmarkInfo.userSavedSearchBookmarkTask?.status ===
          TaskStatus.COMPLETE &&
        this.appBookmarkInfo.userSavedSearchBookmarkTask.data?.id === searchID
      ) {
        // If we already have the data, return it.
        return this.appBookmarkInfo.userSavedSearchBookmarkTask.data;
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
      const q = getSearchQuery(location ?? {search: ''});
      if (this._currentLocation && q && data?.query === q) {
        // Clear out the "q" query parameter if it is the same as the bookmark.
        // Only keep the "q" query parameter if it is different from the bookmark which indicates we are doing an edit.
        updatePageUrl(this._currentLocation.pathname, this._currentLocation, {
          q: '',
        });
      }
      this.refreshAppBookmarkInfo();
    },
    onError: async (error: unknown) => {
      const searchID = this.getSearchID(
        this._currentLocation ?? {search: '', pathname: '', href: ''},
      );
      let err: SavedSearchError;
      if (error instanceof NotFoundError) {
        err = new SavedSearchNotFoundError(searchID);
      } else if (error instanceof ApiError) {
        err = new SavedSearchInternalError(searchID, error.message);
      } else {
        err = new SavedSearchUnknownError(searchID, error);
      }

      this._userSavedBookmarkByIDTaskTracker = {
        status: TaskStatus.ERROR,
        error: err,
        data: undefined,
      };
      this.refreshAppBookmarkInfo();
      if (this._currentLocation) {
        // Clear out the bad "search_id" query parameter.
        updatePageUrl(this._currentLocation.pathname, this._currentLocation, {
          search_id: '',
        });
      }

      // TODO: Reconsider showing the toast in one of the UI components once we have one central
      // UI component that reads the bookmark info instead of the current multiple locations.
      // This will keep the service as purely logical and let the UI component handle the error.
      await new Toast().toast(err.message, 'danger', 'exclamation-triangle');
    },
  });

  _globalBookmarks: Bookmark[];
  _currentGlobalBookmark?: Bookmark;
  // A snapshot of the current location that relates to the bookmark
  // information currently loaded by the service.
  // Typically, we should only update this on navigation events which indicates
  // that we should probably refresh the bookmark information.
  _currentLocation?: AppLocation;

  // Helper for testing.
  getLocation: GetLocationFunction = getCurrentLocation;
  getSearchID: (location: AppLocation) => string = getSearchID;

  constructor() {
    super();
    this._currentLocation = this.getLocation();
    this._globalBookmarks = DEFAULT_BOOKMARKS;
    this._currentGlobalBookmark = this.findCurrentBookmarkByQuery(
      this._globalBookmarks,
    );

    window.addEventListener('popstate', this.handlePopState.bind(this));
  }

  private handlePopState() {
    this._currentLocation = this.getLocation();
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
    const currentQuery = getSearchQuery(this._currentLocation ?? {search: ''});
    return bookmarks?.find(bookmark => bookmark.query === currentQuery);
  }

  // Assign the appBookmarkInfo object to trigger a refresh of subscribed contexts
  refreshAppBookmarkInfo() {
    this.appBookmarkInfo = {
      globalBookmarks: this._globalBookmarks,
      currentGlobalBookmark: this._currentGlobalBookmark,
      userSavedSearchBookmarkTask: this._userSavedBookmarkByIDTaskTracker,
      currentLocation: this._currentLocation,
    };
  }
}
