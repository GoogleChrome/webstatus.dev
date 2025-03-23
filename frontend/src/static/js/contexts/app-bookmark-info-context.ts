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

export interface AppBookmarkInfo {
  globalBookmarks?: Bookmark[];
  currentGlobalBookmark?: Bookmark;
  userSavedSearchBookmarkTask?: TaskTracker<
    Bookmark,
    Error & {message: string}
  >;
}

export function getCurrentBookmark(
  info?: AppBookmarkInfo,
): Bookmark | undefined {
  if (
    info?.userSavedSearchBookmarkTask?.status === TaskStatus.COMPLETE &&
    info?.userSavedSearchBookmarkTask.data
  ) {
    return info.userSavedSearchBookmarkTask.data;
  }

  return info?.currentGlobalBookmark;
}

export const appBookmarkInfoContext =
  createContext<AppBookmarkInfo>('app-bookmark-info');
