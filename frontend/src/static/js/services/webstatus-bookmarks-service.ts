import {customElement} from 'lit/decorators.js';
import {ServiceElement} from './service-element.js';
import {consume, provide} from '@lit/context';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../contexts/app-bookmark-info-context.js';
import {User, firebaseUserContext} from '../contexts/firebase-user-context.js';
import {Task, TaskStatus} from '@lit/task';
import {Bookmark, DEFAULT_BOOKMARKS} from '../utils/constants.js';
import {APIClient, apiClientContext} from '../contexts/api-client-context.js';
import {ApiError, UnknownError} from '../api/errors.js';
import {TaskTracker} from '../utils/task-tracker.js';

@customElement('webstatus-bookmarks-service')
export class WebstatusBookmarksService extends ServiceElement {
  @provide({context: appBookmarkInfoContext})
  appBookmarkInfo: AppBookmarkInfo = {};

  _globalBookmarks: Bookmark[] = DEFAULT_BOOKMARKS;
  _userSavedBookmarksTracker: TaskTracker<Bookmark[], ApiError> = {
    status: TaskStatus.INITIAL,
    data: null,
    error: null,
  };

  @consume({context: firebaseUserContext})
  user?: User;

  @consume({context: apiClientContext})
  apiClient?: APIClient;

  // Assign the appBookmarkInfo object to trigger a refresh of subscribed contexts
  refreshAppBookmarkInfo() {
    this.appBookmarkInfo = {
      globalBookmarks: this._globalBookmarks,
      userSavedBookmarks: this._userSavedBookmarksTracker,
    };
  }

  _task: Task = new Task(this, {
    args: () => [this.user, this.apiClient] as const,
    task: async ([user, apiClient]) => {
      let bookmarks: Bookmark[] | null = null;
      if (!user || !apiClient) {
        this._userSavedBookmarksTracker = {
          status: TaskStatus.INITIAL,
          data: null,
          error: null,
        };
        this.refreshAppBookmarkInfo();
        return bookmarks;
      }

      this._userSavedBookmarksTracker.status = TaskStatus.PENDING;
      this.refreshAppBookmarkInfo();

      const token = await user.getIdToken();
      bookmarks = await apiClient.getAllUserSavedSearches(token);
      return bookmarks;
    },
    onError: async (error: unknown) => {
      let err: ApiError;
      if (error instanceof ApiError) {
        err = error;
      } else {
        // This should not happen but if we don't know the structure of this unknown error,
        // print it out to console and throw an unknown error.
        // The UI will take care of displaying the message to check the console.
        // Print the error to console for debugging purposes.
        console.error(
          `Unknown error fetching saved bookmarks: ${JSON.stringify(error)}`,
        );
        err = new UnknownError(
          "Unknown error fetching saved bookmarks. Please check browser's console for more details.",
        );
      }
      this.appBookmarkInfo = {
        globalBookmarks: this._globalBookmarks,
        userSavedBookmarks: {
          status: TaskStatus.ERROR,
          data: null,
          error: err,
        },
      };
      this.refreshAppBookmarkInfo();
    },
    onComplete: async bookmarks => {
      this.appBookmarkInfo = {
        globalBookmarks: this._globalBookmarks,
        userSavedBookmarks: {
          status: TaskStatus.COMPLETE,
          data: bookmarks,
          error: null,
        },
      };
      this.refreshAppBookmarkInfo();
    },
  });
}
