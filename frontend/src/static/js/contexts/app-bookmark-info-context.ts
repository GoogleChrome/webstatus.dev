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

export interface AppBookmarkInfo {
  globalBookmarks?: Bookmark[];
  currentGlobalBookmark?: Bookmark;
}

export const appBookmarkInfoContext =
  createContext<AppBookmarkInfo>('app-bookmark-info');

/**
 * Returns the current bookmark based on the provided AppBookmarkInfo and location.
 * Currently, it only returns the current global bookmark.
 * In the future, it can be extended to return other bookmarks based on the search parameters.
 *
 * @param {AppBookmarkInfo?} info  - The AppBookmarkInfo object.
 * @param {{search: string}?} _ - The location object containing the search parameters.
 *          Currently not used, but reserved for future use.
 */
export function getCurrentBookmark(
  info?: AppBookmarkInfo,
  _?: {search: string},
): Bookmark | undefined {
  return info?.currentGlobalBookmark;
}
