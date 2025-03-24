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
import {getSearchID} from '../utils/urls.js';

export interface AppBookmarkInfo {
  globalBookmarks?: Bookmark[];
  currentGlobalBookmark?: Bookmark;
  userSavedSearchBookmarkTask?: TaskTracker<Bookmark, SavedSearchError>;
  currentLocation?: {search: string};
}

export function getCurrentBookmark(
  info?: AppBookmarkInfo,
  location?: {search: string},
): Bookmark | undefined {
  const searchID = getSearchID(location ?? {search: ''});
  if (
    // There's a chance that the context has not been updated so we should check the search ID in the location.
    searchID &&
    info?.userSavedSearchBookmarkTask?.status === TaskStatus.COMPLETE &&
    info?.userSavedSearchBookmarkTask.data
  ) {
    return info.userSavedSearchBookmarkTask.data;
  }

  return info?.currentGlobalBookmark;
}
/**
 * Indicates whether the latest bookmark info represents the data found in the current location
 * @param info
 * @param location
 * @returns {boolean}
 */
export function isBookmarkInfoUpdatedToCurrentLocation(
  info?: AppBookmarkInfo,
  location?: {search: string},
): boolean {
  return location?.search === info?.currentLocation?.search;
}

export function isBusyLoadingBookmarkInfo(info?: AppBookmarkInfo): boolean {
  return info?.userSavedSearchBookmarkTask?.status === TaskStatus.PENDING;
}

export function isSavedSearchNotFound(info?: AppBookmarkInfo): boolean {
  return (
    info?.userSavedSearchBookmarkTask?.status === TaskStatus.ERROR &&
    info.userSavedSearchBookmarkTask.error instanceof SavedSearchNotFoundError
  );
}

export const appBookmarkInfoContext =
  createContext<AppBookmarkInfo>('app-bookmark-info');

export type SavedSearchError = Error & {message: string};

export class SavedSearchNotFoundError extends Error {
  constructor(id: string) {
    super(`Saved search with id ${id} not found`);
  }
}

export class SavedSearchInternalError extends Error {
  constructor(id: string, msg: string) {
    super(`Error fetching saved search ID ${id}: ${msg}`);
  }
}

export class SavedSearchUnknownError extends Error {
  constructor(id: string, err: unknown) {
    super(
      `Unknown error fetching saved search ID ${id}. Check console for details.`,
    );
    console.error(err);
  }
}
