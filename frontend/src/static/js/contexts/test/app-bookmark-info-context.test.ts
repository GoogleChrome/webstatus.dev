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

import {
  savedSearchHelpers,
  SavedSearchNotFoundError,
  SavedSearchInternalError,
  SavedSearchUnknownError,
  AppBookmarkInfo,
  UserSavedSearchesInternalError,
  UserSavedSearchesUnknownError,
  CurrentSavedSearch,
  SavedSearchScope,
} from '../app-bookmark-info-context.js';
import {TaskStatus} from '@lit/task';
import {expect} from '@open-wc/testing';
import sinon from 'sinon';

describe('app-bookmark-info-context', () => {
  describe('savedSearchHelpers', () => {
    describe('getCurrentSavedSearch', () => {
      it('should return undefined if no info is provided', () => {
        expect(savedSearchHelpers.getCurrentSavedSearch()).to.be.undefined;
      });

      it('should return the userSavedSearchTask data if available and complete', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(info, {
            search: '?search_id=123',
          }),
        ).to.deep.equal({
          scope: SavedSearchScope.UserSavedSearch,
          value: {
            query: 'test',
            name: 'Test Bookmark',
            id: '123',
          },
        });
      });

      it('should return the currentGlobalSavedSearch if userSavedSearchTask is not complete', () => {
        const expectedData: CurrentSavedSearch = {
          scope: SavedSearchScope.GlobalSavedSearch,
          value: {
            query: 'global',
            name: 'Global Bookmark',
          },
        };
        // Pending state
        const pendingInfo: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(pendingInfo),
        ).to.deep.equal(expectedData);
        // Initial state
        const initialInfo: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(initialInfo),
        ).to.deep.equal(expectedData);
        // Undefined state
        const undefinedInfo: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchTask: undefined,
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(undefinedInfo),
        ).to.deep.equal(expectedData);
      });

      it('should return undefined if no bookmark is found', () => {
        const info: AppBookmarkInfo = {};
        expect(savedSearchHelpers.getCurrentSavedSearch(info)).to.be.undefined;
      });

      it('should handle a location with a search ID', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        const location = {search: '?search_id=123'};
        expect(
          savedSearchHelpers.getCurrentSavedSearch(info, location),
        ).to.deep.equal({
          scope: SavedSearchScope.UserSavedSearch,
          value: {query: 'test', name: 'Test Bookmark', id: '123'},
        });
      });

      it('should return the data from userSavedSearchesTask if available and complete', () => {
        const info = {
          userSavedSearchesTask: {
            status: TaskStatus.COMPLETE,
            data: [{query: 'test', name: 'Test Bookmark', id: '123'}],
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(info, {
            search: '?search_id=123',
          }),
        ).to.deep.equal({
          scope: SavedSearchScope.UserSavedSearch,
          value: {query: 'test', name: 'Test Bookmark', id: '123'},
        });
      });

      it('should return undefined if userSavedSearchesTask is not complete', () => {
        const info = {
          userSavedSearchesTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(savedSearchHelpers.getCurrentSavedSearch(info)).to.be.undefined;
      });

      it('should return undefined if userSavedSearchesTask data is empty', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchesTask: {
            status: TaskStatus.COMPLETE,
            data: [],
            error: undefined,
          },
        };
        expect(savedSearchHelpers.getCurrentSavedSearch(info)).to.be.undefined;
      });

      it('should return the currentGlobalSavedSearch if userSavedSearchesTask is not complete', () => {
        const expectedData: CurrentSavedSearch = {
          scope: SavedSearchScope.GlobalSavedSearch,
          value: {
            query: 'global',
            name: 'Global Bookmark',
          },
        };
        // Pending state
        const pendingInfo: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchesTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(pendingInfo),
        ).to.deep.equal(expectedData);
        // Initial state
        const initialInfo: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchesTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(initialInfo),
        ).to.deep.equal(expectedData);
        // Undefined state
        const undefinedInfo: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchesTask: undefined,
        };
        expect(
          savedSearchHelpers.getCurrentSavedSearch(undefinedInfo),
        ).to.deep.equal(expectedData);
      });
    });

    describe('getCurrentQuery', () => {
      it('should return the query from the location if info is undefined', () => {
        const location = {search: '?q=test'};
        expect(
          savedSearchHelpers.getCurrentQuery(undefined, location),
        ).to.equal('test');
      });

      it('should return the query from the location if bookmark info is loading', () => {
        const location = {search: '?q=test'};
        const expectedQuery = 'test';
        // Pending
        const pendingInfo: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
          userSavedSearchesTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentQuery(pendingInfo, location),
        ).to.equal(expectedQuery);
        // Initial
        const initialInfo: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
          userSavedSearchesTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.getCurrentQuery(initialInfo, location),
        ).to.equal(expectedQuery);
        // Undefined
        const undefinedInfo: AppBookmarkInfo = {
          userSavedSearchTask: undefined,
          userSavedSearchesTask: undefined,
        };
        expect(
          savedSearchHelpers.getCurrentQuery(undefinedInfo, location),
        ).to.equal(expectedQuery);
      });

      it('should return the query from the userSavedSearchTask if available and complete', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        const isBusyStub = sinon.stub(
          savedSearchHelpers,
          'isBusyLoadingSavedSearchInfo',
        );
        isBusyStub.returns(false);
        const getCurrentBookmarkStub = sinon.stub(
          savedSearchHelpers,
          'getCurrentSavedSearch',
        );
        getCurrentBookmarkStub.returns({
          scope: SavedSearchScope.UserSavedSearch,
          value: {
            query: 'test',
            name: 'Test Bookmark',
            id: '123',
          },
        });
        expect(
          savedSearchHelpers.getCurrentQuery(info, {search: '?search_id=123'}),
        ).to.equal('test');
        getCurrentBookmarkStub.restore();
        isBusyStub.restore();
      });

      it('should return the query from the currentGlobalSavedSearch if available', () => {
        const info: AppBookmarkInfo = {
          currentGlobalSavedSearch: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchTask: {
            status: TaskStatus.COMPLETE,
            data: undefined,
            error: undefined,
          },
          userSavedSearchesTask: {
            status: TaskStatus.COMPLETE,
            data: undefined,
            error: undefined,
          },
        };
        expect(savedSearchHelpers.getCurrentQuery(info)).to.equal('global');
      });

      it('should return an empty string if no query is found', () => {
        expect(savedSearchHelpers.getCurrentQuery()).to.equal('');
      });

      it('should prioritize q parameter over bookmark query if different', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        const location = {search: '?q=edited'};
        expect(savedSearchHelpers.getCurrentQuery(info, location)).to.equal(
          'edited',
        );
      });
    });

    describe('isBusyLoadingSavedSearchInfo', () => {
      it('should return true if userSavedSearchTask is pending/initial/undefined', () => {
        const pendingInfo: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.isBusyLoadingSavedSearchInfo(pendingInfo),
        ).to.equal(true);
        const initialInfo: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(
          savedSearchHelpers.isBusyLoadingSavedSearchInfo(initialInfo),
        ).to.equal(true);
        const undefinedInfo: AppBookmarkInfo = {
          userSavedSearchTask: undefined,
        };
        expect(
          savedSearchHelpers.isBusyLoadingSavedSearchInfo(undefinedInfo),
        ).to.equal(true);
      });

      it('should return true if currentLocation search is different from location search', () => {
        const info: AppBookmarkInfo = {currentLocation: {search: '?q=old'}};
        const location = {search: '?q=new'};
        expect(
          savedSearchHelpers.isBusyLoadingSavedSearchInfo(info, location),
        ).to.equal(true);
      });

      it('should return false if userSavedSearchTask is complete and locations match', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchTask: {
            status: TaskStatus.COMPLETE,
            data: undefined,
            error: undefined,
          },
          userSavedSearchesTask: {
            status: TaskStatus.COMPLETE,
            data: undefined,
            error: undefined,
          },
          currentLocation: {search: '?q=test'},
        };
        const location = {search: '?q=test'};
        expect(
          savedSearchHelpers.isBusyLoadingSavedSearchInfo(info, location),
        ).to.equal(false);
      });
    });
  });

  describe('Error Classes', () => {
    it('SavedSearchNotFoundError should create correct error message', () => {
      const error = new SavedSearchNotFoundError('123');
      expect(error.message).to.equal('Saved search with id 123 not found');
    });

    it('SavedSearchInternalError should create correct error message', () => {
      const error = new SavedSearchInternalError('123', 'Server Error');
      expect(error.message).to.equal(
        'Error fetching saved search ID 123: Server Error',
      );
    });

    it('SavedSearchUnknownError should create correct error message and log error to console', () => {
      const error = new SavedSearchUnknownError(
        '123',
        new Error('Saved Search Unknown Test Error'),
      );
      expect(error.message).to.equal(
        'Unknown error fetching saved search ID 123. Check console for details.',
      );
    });

    it('UserSavedSearchesInternalError should create correct error message', () => {
      const error = new UserSavedSearchesInternalError('Server Error');
      expect(error.message).to.equal(
        'Internal error fetching list of saved searches for user: Server Error',
      );
    });

    it('UserSavedSearchesUnknownError should create correct error message and log error to console', () => {
      const error = new UserSavedSearchesUnknownError(
        new Error('User Saved Searches Unknown Test Error'),
      );
      expect(error.message).to.equal(
        'Unknown error fetching list of saved searches for user. Check console for details.',
      );
    });
  });
});
