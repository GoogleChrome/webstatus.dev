import {createContext} from '@lit/context';
import {Bookmark} from '../utils/constants.js';
import {TaskTracker} from '../utils/task-tracker.js';
import {ApiError} from '../api/errors.js';

export interface AppBookmarkInfo {
  globalBookmarks?: Bookmark[];
  userSavedBookmarks?: TaskTracker<Bookmark[], ApiError>;
}

export const appBookmarkInfoContext =
  createContext<AppBookmarkInfo>('app-bookmark-info');
