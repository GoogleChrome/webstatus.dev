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
import {getSearchQuery} from '../utils/urls.js';

export interface AppBookmarkInfo {
  globalSavedSearches?: GlobalSavedSearch[];
  currentGlobalSavedSearch?: GlobalSavedSearch;
  userSavedSearchTask?: TaskTracker<UserSavedSearch, SavedSearchError>;
  userSavedSearchesTask?: TaskTracker<UserSavedSearch[], SavedSearchError>;
  globalSavedSearchesTask?: TaskTracker<GlobalSavedSearch[], Error>;
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
    const q = getSearchQuery(info?.currentLocation ?? {search: ''});
    const trimmed = q.trim();

    // Check Global Saved Searches
    if (trimmed.startsWith('hotlist:')) {
      const parts = trimmed.split(':');
      if (parts.length === 2 && info?.globalSavedSearches) {
        let raw = parts[1];
        if (raw.startsWith('"') && raw.endsWith('"')) raw = raw.slice(1, -1);
        const globalMatch = info.globalSavedSearches.find(s => s.id === raw);
        if (globalMatch)
          return {
            scope: SavedSearchScope.GlobalSavedSearch,
            value: globalMatch,
          };
      }
    }

    // Check User Saved Searches
    if (trimmed.startsWith('saved:')) {
      const parts = trimmed.split(':');
      if (parts.length === 2) {
        let raw = parts[1];
        if (raw.startsWith('"') && raw.endsWith('"')) raw = raw.slice(1, -1);

        if (
          info?.userSavedSearchesTask?.status === TaskStatus.COMPLETE &&
          info?.userSavedSearchesTask.data
        ) {
          const userMatch = info.userSavedSearchesTask.data.find(
            s => s.id === raw,
          );
          if (userMatch)
            return {scope: SavedSearchScope.UserSavedSearch, value: userMatch};
        }
        if (
          info?.userSavedSearchTask?.status === TaskStatus.COMPLETE &&
          info?.userSavedSearchTask.data
        ) {
          if (info.userSavedSearchTask.data.id === raw) {
            return {
              scope: SavedSearchScope.UserSavedSearch,
              value: info.userSavedSearchTask.data,
            };
          }
        }
      }
    }

    // Fallback for legacy exact query string match
    const currentGlobal = info?.currentGlobalSavedSearch;
    if (currentGlobal && currentGlobal.query === q) {
      return {scope: SavedSearchScope.GlobalSavedSearch, value: currentGlobal};
    }

    return undefined;
  },

  getCurrentQuery: (info?: AppBookmarkInfo): string => {
    const currentLocation = info?.currentLocation ?? {search: ''};
    const q = getSearchQuery(currentLocation);
    const trimmed = q.trim();
    if (trimmed.startsWith('saved:')) {
      const search = savedSearchHelpers.getCurrentSavedSearch(info);
      if (search) {
        return search.value.query;
      }
    }
    return q;
  },

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
  // eslint-disable-next-line @typescript-eslint/no-restricted-types
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
  // eslint-disable-next-line @typescript-eslint/no-restricted-types
  constructor(err: unknown) {
    super(
      'Unknown error fetching list of saved searches for user. Check console for details.',
    );
    console.error(err);
  }
}
