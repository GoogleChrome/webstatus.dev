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

import {createContext} from '@lit/context';
import {GlobalSavedSearch, UserSavedSearch} from '../utils/constants.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {TaskStatus} from '@lit/task';
import {getSearchID, getSearchQuery} from '../utils/urls.js';

export interface AppBookmarkInfo {
  globalSavedSearches?: GlobalSavedSearch[];
  currentGlobalSavedSearch?: GlobalSavedSearch;
  userSavedSearchTask?: TaskTracker<UserSavedSearch, SavedSearchError>;
  userSavedSearchesTask?: TaskTracker<UserSavedSearch[], SavedSearchError>;
  currentLocation?: {search: string};
}

export const appBookmarkInfoContext =
  createContext<AppBookmarkInfo>('app-bookmark-info');

export enum SavedSearchScope {
  GlobalSavedSearch,
  UserSavedSearch,
}

export type CurrentSavedSearch =
  | {scope: SavedSearchScope.GlobalSavedSearch; value: GlobalSavedSearch}
  | {scope: SavedSearchScope.UserSavedSearch; value: UserSavedSearch}
  | undefined;
export const savedSearchHelpers = {
  /**
   * Returns the current saved search based on the provided AppBookmarkInfo.
   *
   * @param {AppBookmarkInfo?} info  - The AppBookmarkInfo object.
   */
  getCurrentSavedSearch(info?: AppBookmarkInfo): CurrentSavedSearch {
    const searchID = getSearchID(info?.currentLocation ?? {search: ''});
    if (
      // There's a chance that the context has not been updated so we should check the search ID in the location.
      searchID &&
      info?.userSavedSearchesTask?.status === TaskStatus.COMPLETE &&
      info?.userSavedSearchesTask.data
    ) {
      const userSavedSearch = info.userSavedSearchesTask.data?.find(
        item => item.id === searchID,
      );
      if (userSavedSearch !== undefined) {
        return {
          scope: SavedSearchScope.UserSavedSearch,
          value: userSavedSearch,
        };
      }
    }
    if (
      // There's a chance that the context has not been updated so we should check the search ID in the location.
      searchID &&
      info?.userSavedSearchTask?.status === TaskStatus.COMPLETE &&
      info?.userSavedSearchTask.data
    ) {
      return {
        scope: SavedSearchScope.UserSavedSearch,
        value: info.userSavedSearchTask.data,
      };
    }

    if (info?.currentGlobalSavedSearch) {
      return {
        scope: SavedSearchScope.GlobalSavedSearch,
        value: info.currentGlobalSavedSearch,
      };
    }

    return undefined;
  },

  /**
   * Returns the current query based on the provided AppBookmarkInfo.
   *
   * This function determines the active query string by considering both global
   * and user saved searches, as well as the current location's search parameters.
   *
   * - If a user saved search is active (indicated by a matching `search_id` in
   *   the location), its query is used unless the location's `q` parameter is
   *   different, which indicates the user is editing the query.
   * - If a global saved search is active, its query is used.
   * - If no saved search is active, the query from the location's `q` parameter is used.
   * - If the saved search information is still loading, the query from the location's `q` parameter is used.
   *
   * @param {AppBookmarkInfo?} info - The AppBookmarkInfo object.
   * @returns {string} The current query string.
   */
  getCurrentQuery: (info?: AppBookmarkInfo): string => {
    const currentLocation = info?.currentLocation ?? {search: ''};
    const q = getSearchQuery(currentLocation);
    if (savedSearchHelpers.isBusyLoadingSavedSearchInfo(info)) {
      return q;
    }
    const savedSearch = savedSearchHelpers.getCurrentSavedSearch(info);
    // User saved searches can be edited. And those have IDs
    if (savedSearch?.scope === SavedSearchScope.UserSavedSearch) {
      // If there's a saved search, prioritize its query unless q is different.
      // If they are different, this could mean we are trying to edit.
      return q !== savedSearch.value.query && q !== ''
        ? q
        : savedSearch.value.query;
    } else if (savedSearch?.scope === SavedSearchScope.GlobalSavedSearch) {
      // If there's a global saved search, use its query.
      return savedSearch.value.query;
    }

    return q;
  },

  /**
   * Checks if the bookmark information is currently being loaded.
   *
   * @param {AppBookmarkInfo?} info - The AppBookmarkInfo object.
   * @returns {boolean} True if the bookmark info is loading or the location has changed, false otherwise.
   */
  isBusyLoadingSavedSearchInfo: (info?: AppBookmarkInfo): boolean => {
    return (
      info?.userSavedSearchTask === undefined ||
      info?.userSavedSearchTask?.status === TaskStatus.INITIAL ||
      info?.userSavedSearchTask?.status === TaskStatus.PENDING ||
      info?.userSavedSearchesTask === undefined ||
      info?.userSavedSearchesTask?.status === TaskStatus.INITIAL ||
      info?.userSavedSearchesTask?.status === TaskStatus.PENDING
    );
  },
};

/**
 * Represents an error related to saved searches.
 */
export type SavedSearchError = Error & {message: string};

/**
 * Represents an error when a saved search is not found.
 */
export class SavedSearchNotFoundError extends Error {
  /**
   * Creates a new SavedSearchNotFoundError.
   * @param {string} id - The ID of the saved search that was not found.
   */
  constructor(id: string) {
    super(`Saved search with id ${id} not found`);
  }
}

/**
 * Represents an internal error that occurred while fetching a saved search.
 */
export class SavedSearchInternalError extends Error {
  /**
   * Creates a new SavedSearchInternalError.
   * @param {string} id - The ID of the saved search that caused the error.
   * @param {string} msg - The error message.
   */
  constructor(id: string, msg: string) {
    super(`Error fetching saved search ID ${id}: ${msg}`);
  }
}

/**
 * Represents an unknown error that occurred while fetching a saved search.
 */
export class SavedSearchUnknownError extends Error {
  /**
   * Creates a new SavedSearchUnknownError.
   * @param {string} id - The ID of the saved search that caused the error.
   * @param {unknown} err - The unknown error.
   */
  constructor(id: string, err: unknown) {
    super(
      `Unknown error fetching saved search ID ${id}. Check console for details.`,
    );
    console.error(err);
  }
}

/**
 * Represents an internal error that occurred while fetching a user's list of saved searches.
 */
export class UserSavedSearchesInternalError extends Error {
  /**
   * Creates a new UserSavedSearchesInternalError.
   * @param {string} msg - The error message.
   */
  constructor(msg: string) {
    super(`Internal error fetching list of saved searches for user: ${msg}`);
  }
}

/**
 * Represents an unknown error that occurred while fetching a user's list of saved searches.
 */
export class UserSavedSearchesUnknownError extends Error {
  /**
   * Creates a new UserSavedSearchesUnknownError.
   * @param {unknown} err - The unknown error.
   */
  constructor(err: unknown) {
    super(
      'Unknown error fetching list of saved searches for user. Check console for details.',
    );
    console.error(err);
  }
}
