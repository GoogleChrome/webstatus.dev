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
import {Bookmark} from '../utils/constants.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {TaskStatus} from '@lit/task';
import {getSearchID, getSearchQuery} from '../utils/urls.js';

export interface AppBookmarkInfo {
  globalBookmarks?: Bookmark[];
  currentGlobalBookmark?: Bookmark;
  userSavedSearchBookmarkTask?: TaskTracker<Bookmark, SavedSearchError>;
  userSavedSearchBookmarksTask?: TaskTracker<Bookmark[], SavedSearchError>;
  currentLocation?: {search: string};
}

export const appBookmarkInfoContext =
  createContext<AppBookmarkInfo>('app-bookmark-info');

export const bookmarkHelpers = {
  /**
   * Returns the current bookmark based on the provided AppBookmarkInfo and location.
   *
   * @param {AppBookmarkInfo?} info  - The AppBookmarkInfo object.
   * @param {{search: string}?} location - The location object containing the search parameters.
   */
  getCurrentBookmark(
    info?: AppBookmarkInfo,
    location?: {search: string},
  ): Bookmark | undefined {
    const searchID = getSearchID(location ?? {search: ''});
    if (
      // There's a chance that the context has not been updated so we should check the search ID in the location.
      searchID &&
      info?.userSavedSearchBookmarksTask?.status === TaskStatus.COMPLETE &&
      info?.userSavedSearchBookmarksTask.data
    ) {
      const userBookmark = info.userSavedSearchBookmarksTask.data?.find(
        item => item.id === searchID,
      );
      if (userBookmark !== undefined) {
        return userBookmark;
      }
    }
    if (
      // There's a chance that the context has not been updated so we should check the search ID in the location.
      searchID &&
      info?.userSavedSearchBookmarkTask?.status === TaskStatus.COMPLETE &&
      info?.userSavedSearchBookmarkTask.data
    ) {
      return info.userSavedSearchBookmarkTask.data;
    }

    return info?.currentGlobalBookmark;
  },

  /**
   * Returns the current query based on the provided AppBookmarkInfo and location.
   *
   * This function determines the active query string by considering both global
   * and user-saved bookmarks, as well as the current location's search parameters.
   *
   * - If a user-saved bookmark is active (indicated by a matching `search_id` in
   *   the location), its query is used unless the location's `q` parameter is
   *   different, which indicates the user is editing the query.
   * - If a global bookmark is active, its query is used.
   * - If no bookmark is active, the query from the location's `q` parameter is used.
   * - If the bookmark information is still loading, the query from the location's `q` parameter is used.
   *
   * @param {AppBookmarkInfo?} info - The AppBookmarkInfo object.
   * @param {{search: string}?} location - The location object containing the search parameters.
   * @returns {string} The current query string.
   */
  getCurrentQuery: (
    info?: AppBookmarkInfo,
    location?: {search: string},
  ): string => {
    const q = getSearchQuery(location ?? {search: ''});
    if (bookmarkHelpers.isBusyLoadingBookmarkInfo(info, location)) {
      return q;
    }
    const bookmark = bookmarkHelpers.getCurrentBookmark(info, location);
    // User saved bookmarks can be edited. And those have IDs
    if (bookmark !== undefined && bookmark.id !== undefined) {
      // If there's a bookmark, prioritize its query unless q is different.
      // If they are different, this could mean we are trying to edit.
      return q !== bookmark.query && q !== '' ? q : bookmark.query;
    } else if (bookmark !== undefined && bookmark.id === undefined) {
      // If there's a global bookmark, use its query.
      return bookmark.query;
    }

    return q;
  },

  /**
   * Checks if the bookmark information is currently being loaded or if the
   * current location has changed.
   *
   * @param {AppBookmarkInfo?} info - The AppBookmarkInfo object.
   * @param {{search: string}?} location - The location object containing the search parameters.
   * @returns {boolean} True if the bookmark info is loading or the location has changed, false otherwise.
   */
  isBusyLoadingBookmarkInfo: (
    info?: AppBookmarkInfo,
    location?: {search: string},
  ): boolean => {
    return (
      info?.userSavedSearchBookmarkTask === undefined ||
      info?.userSavedSearchBookmarkTask?.status === TaskStatus.INITIAL ||
      info?.userSavedSearchBookmarkTask?.status === TaskStatus.PENDING ||
      info?.userSavedSearchBookmarksTask === undefined ||
      info?.userSavedSearchBookmarksTask?.status === TaskStatus.INITIAL ||
      info?.userSavedSearchBookmarksTask?.status === TaskStatus.PENDING ||
      info?.currentLocation?.search !== location?.search
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
 * Represents an internal error that occurred while fetching a user's list of bookmaked saved searches.
 */
export class UserSavedSearchesInternalError extends Error {
  /**
   * Creates a new UserSavedSearchesInternalError.
   * @param {string} msg - The error message.
   */
  constructor(msg: string) {
    super(
      `Internal error fetching list of bookmarked saved searches for user: ${msg}`,
    );
  }
}

/**
 * Represents an unknown error that occurred while fetching a user's list of bookmaked saved searches.
 */
export class UserSavedSearchesUnknownError extends Error {
  /**
   * Creates a new UserSavedSearchesUnknownError.
   * @param {unknown} err - The unknown error.
   */
  constructor(err: unknown) {
    super(
      'Unknown error fetching list of bookmarked saved searches for user. Check console for details.',
    );
    console.error(err);
  }
}
