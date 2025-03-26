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
  UserSavedSearchesInternalError,
  UserSavedSearchesUnknownError,
  appBookmarkInfoContext,
} from '../contexts/app-bookmark-info-context.js';
import {Bookmark, DEFAULT_BOOKMARKS} from '../utils/constants.js';
import {
  QueryStringOverrides,
  getSearchID,
  getSearchQuery,
  updatePageUrl,
} from '../utils/urls.js';
import {AppLocation, getCurrentLocation} from '../utils/app-router.js';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {User} from 'firebase/auth';
import {firebaseUserContext} from '../contexts/firebase-user-context.js';
import {Task, TaskStatus} from '@lit/task';
import {NotFoundError, ApiError} from '../api/errors.js';
import {
  DuplicateTaskFailedError,
  DuplicateTaskPendingError,
  TaskNotReadyError,
  TaskTracker,
} from '../utils/task-tracker.js';
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
  user: User | null | undefined;

  _userSavedBookmarkByIDTaskTracker?: TaskTracker<
    Bookmark,
    SavedSearchError
  > & {taskLocation: AppLocation} = undefined;

  _userSavedBookmarksTaskTracker?: TaskTracker<Bookmark[], SavedSearchError> =
    undefined;

  loadingUserSavedBookmarkByIDTask = new Task(this, {
    args: () => [this._currentLocation, this.apiClient, this.user] as const,
    task: async ([currentLocation, apiClient, user]) => {
      if (!apiClient || !currentLocation) {
        throw new TaskNotReadyError();
      }
      // Derive a unique search ID from the current location. This ID is used
      // to identify the specific bookmark being requested.
      const searchID = this.getSearchID(currentLocation);
      if (!searchID) {
        throw new TaskNotReadyError();
      }

      // Get the search ID of the previously executed task (if any).
      const previousTaskSearchID = this.getSearchID(
        this._userSavedBookmarkByIDTaskTracker?.taskLocation ?? {
          search: '',
          pathname: '',
          href: '',
        },
      );

      // Check if the current request is for the same bookmark (same searchID)
      // as a previously initiated task.
      if (searchID === previousTaskSearchID) {
        // If the previous task was successful, return the cached data to avoid
        // redundant API calls.
        switch (this._userSavedBookmarkByIDTaskTracker?.status) {
          case TaskStatus.COMPLETE:
            return {
              search: this._userSavedBookmarkByIDTaskTracker.data,
              taskLocation:
                this._userSavedBookmarkByIDTaskTracker?.taskLocation,
            };
          case TaskStatus.PENDING:
            throw new DuplicateTaskPendingError();
          case TaskStatus.ERROR:
            throw new DuplicateTaskFailedError();
          default:
            // Fallback to DuplicateTaskPendingError. Should not get here.
            throw new DuplicateTaskPendingError();
        }
      }

      let token: string | undefined;
      if (user) {
        token = await user.getIdToken();
      }
      this._userSavedBookmarkByIDTaskTracker = {
        status: TaskStatus.PENDING,
        data: undefined,
        error: undefined,
        taskLocation: currentLocation,
      };
      this.refreshAppBookmarkInfo();

      const savedSearch = await apiClient.getSavedSearchByID(searchID, token);
      return {search: savedSearch, taskLocation: currentLocation};
    },
    onComplete: data => {
      const taskLocation = data?.taskLocation ?? {
        search: '',
        pathname: '',
        href: '',
      };
      this._userSavedBookmarkByIDTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: data.search,
        error: undefined,
        taskLocation: data.taskLocation,
      };

      const q = getSearchQuery(taskLocation ?? {search: ''});
      if (taskLocation && q && data.search?.query === q) {
        // Clear out the "q" query parameter if it is the same as the bookmark.
        // Only keep the "q" query parameter if it is different from the bookmark which indicates we are doing an edit.
        this.updatePageUrl(taskLocation.pathname, taskLocation, {
          q: '',
        });
      }
      this.refreshAppBookmarkInfo();
    },
    onError: async (error: unknown) => {
      if (
        error instanceof TaskNotReadyError ||
        error instanceof DuplicateTaskFailedError ||
        error instanceof DuplicateTaskPendingError
      ) {
        // Don't touch the task tracker
        return;
      }
      const taskLocation = this._userSavedBookmarkByIDTaskTracker
        ?.taskLocation ?? {search: '', pathname: '', href: ''};
      const searchID = this.getSearchID(taskLocation);
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
        taskLocation: this._userSavedBookmarkByIDTaskTracker?.taskLocation ?? {
          search: '',
          pathname: '',
          href: '',
        },
      };
      this.refreshAppBookmarkInfo();
      // Clear out the bad "search_id" query parameter.
      this.updatePageUrl(taskLocation.pathname, taskLocation, {
        search_id: '',
      });

      // TODO: Reconsider showing the toast in one of the UI components once we have one central
      // UI component that reads the bookmark info instead of the current multiple locations.
      // This will keep the service as purely logical and let the UI component handle the error.
      await new Toast().toast(err.message, 'danger', 'exclamation-triangle');
    },
  });

  loadingUserSavedBookmarksTask = new Task(this, {
    args: () => [this.apiClient, this.user] as const,
    task: async ([apiClient, user]) => {
      if (user === undefined || !apiClient) {
        this._userSavedBookmarksTaskTracker = {
          status: TaskStatus.PENDING,
          data: undefined,
          error: undefined,
        };
        this.refreshAppBookmarkInfo();
        throw new TaskNotReadyError();
      }
      if (user === null) {
        return undefined;
      }
      const token = await user.getIdToken();
      this._userSavedBookmarksTaskTracker = {
        status: TaskStatus.PENDING,
        data: undefined,
        error: undefined,
      };
      this.refreshAppBookmarkInfo();

      return await apiClient.getAllUserSavedSearches(token);
    },
    onComplete: data => {
      this._userSavedBookmarksTaskTracker = {
        status: TaskStatus.COMPLETE,
        data: data,
        error: undefined,
      };
      this.refreshAppBookmarkInfo();
    },
    onError: async (error: unknown) => {
      if (error instanceof TaskNotReadyError) {
        // Don't touch the task tracker
        return;
      }

      let err: SavedSearchError;

      if (error instanceof ApiError) {
        err = new UserSavedSearchesInternalError(error.message);
      } else {
        err = new UserSavedSearchesUnknownError(error);
      }

      this._userSavedBookmarksTaskTracker = {
        status: TaskStatus.ERROR,
        error: err,
        data: undefined,
      };
      this.refreshAppBookmarkInfo();

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
  updatePageUrl: (
    pathname: string,
    location: {search: string},
    overrides: QueryStringOverrides,
  ) => void = updatePageUrl;

  constructor() {
    super();
    this._globalBookmarks = DEFAULT_BOOKMARKS;
    this._currentGlobalBookmark = this.findCurrentBookmarkByQuery(
      this._globalBookmarks,
    );
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
    this._currentLocation = this.getLocation();
    window.addEventListener('popstate', this.handlePopState.bind(this));
    this.refreshAppBookmarkInfo();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.removeEventListener('popstate', this.handlePopState.bind(this));
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
      currentLocation: this._currentLocation,
      userSavedSearchBookmarksTask: this._userSavedBookmarksTaskTracker,
      // Exclude the taskLocation property from the tracker object
      userSavedSearchBookmarkTask: this._userSavedBookmarkByIDTaskTracker
        ? {
            status: this._userSavedBookmarkByIDTaskTracker?.status,
            data: this._userSavedBookmarkByIDTaskTracker?.data,
            error: this._userSavedBookmarkByIDTaskTracker?.error,
          }
        : undefined,
    };
  }
}
