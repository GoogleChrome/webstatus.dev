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
  bookmarkHelpers,
  SavedSearchNotFoundError,
  SavedSearchInternalError,
  SavedSearchUnknownError,
  AppBookmarkInfo,
} from '../app-bookmark-info-context.js';
import {TaskStatus} from '@lit/task';
import {expect} from '@open-wc/testing';
import sinon from 'sinon';

describe('app-bookmark-info-context', () => {
  describe('bookmarkHelpers', () => {
    describe('getCurrentBookmark', () => {
      it('should return undefined if no info is provided', () => {
        expect(bookmarkHelpers.getCurrentBookmark()).to.be.undefined;
      });

      it('should return the userSavedSearchBookmarkTask data if available and complete', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        expect(
          bookmarkHelpers.getCurrentBookmark(info, {search: '?search_id=123'}),
        ).to.deep.equal({
          query: 'test',
          name: 'Test Bookmark',
          id: '123',
        });
      });

      it('should return the currentGlobalBookmark if userSavedSearchBookmarkTask is not complete', () => {
        const expectedData = {
          query: 'global',
          name: 'Global Bookmark',
        };
        // Pending state
        const pendingInfo: AppBookmarkInfo = {
          currentGlobalBookmark: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchBookmarkTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.getCurrentBookmark(pendingInfo)).to.deep.equal(
          expectedData,
        );
        // Initial state
        const initialInfo: AppBookmarkInfo = {
          currentGlobalBookmark: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchBookmarkTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.getCurrentBookmark(initialInfo)).to.deep.equal(
          expectedData,
        );
        // Undefined state
        const undefinedInfo: AppBookmarkInfo = {
          currentGlobalBookmark: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchBookmarkTask: undefined,
        };
        expect(bookmarkHelpers.getCurrentBookmark(undefinedInfo)).to.deep.equal(
          expectedData,
        );
      });

      it('should return undefined if no bookmark is found', () => {
        const info: AppBookmarkInfo = {};
        expect(bookmarkHelpers.getCurrentBookmark(info)).to.be.undefined;
      });

      it('should handle a location with a search ID', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        const location = {search: '?search_id=123'};
        expect(
          bookmarkHelpers.getCurrentBookmark(info, location),
        ).to.deep.equal({query: 'test', name: 'Test Bookmark', id: '123'});
      });
    });

    describe('getCurrentQuery', () => {
      it('should return the query from the location if info is undefined', () => {
        const location = {search: '?q=test'};
        expect(bookmarkHelpers.getCurrentQuery(undefined, location)).to.equal(
          'test',
        );
      });

      it('should return the query from the location if bookmark info is loading', () => {
        const location = {search: '?q=test'};
        const expectedQuery = 'test';
        // Pending
        const pendingInfo: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.getCurrentQuery(pendingInfo, location)).to.equal(
          expectedQuery,
        );
        // Initial
        const initialInfo: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.getCurrentQuery(initialInfo, location)).to.equal(
          expectedQuery,
        );
        // Undefined
        const undefinedInfo: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: undefined,
        };
        expect(
          bookmarkHelpers.getCurrentQuery(undefinedInfo, location),
        ).to.equal(expectedQuery);
      });

      it('should return the query from the userSavedSearchBookmarkTask if available and complete', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        const isBusyStub = sinon.stub(
          bookmarkHelpers,
          'isBusyLoadingBookmarkInfo',
        );
        isBusyStub.returns(false);
        const getCurrentBookmarkStub = sinon.stub(
          bookmarkHelpers,
          'getCurrentBookmark',
        );
        getCurrentBookmarkStub.returns({
          query: 'test',
          name: 'Test Bookmark',
          id: '123',
        });
        expect(
          bookmarkHelpers.getCurrentQuery(info, {search: '?search_id=123'}),
        ).to.equal('test');
        getCurrentBookmarkStub.restore();
        isBusyStub.restore();
      });

      it('should return the query from the currentGlobalBookmark if available', () => {
        const info: AppBookmarkInfo = {
          currentGlobalBookmark: {
            query: 'global',
            name: 'Global Bookmark',
          },
          userSavedSearchBookmarkTask: {
            status: TaskStatus.COMPLETE,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.getCurrentQuery(info)).to.equal('global');
      });

      it('should return an empty string if no query is found', () => {
        expect(bookmarkHelpers.getCurrentQuery()).to.equal('');
      });

      it('should prioritize q parameter over bookmark query if different', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.COMPLETE,
            data: {query: 'test', name: 'Test Bookmark', id: '123'},
            error: undefined,
          },
        };
        const location = {search: '?q=edited'};
        expect(bookmarkHelpers.getCurrentQuery(info, location)).to.equal(
          'edited',
        );
      });
    });

    describe('isBusyLoadingBookmarkInfo', () => {
      it('should return true if userSavedSearchBookmarkTask is pending/initial/undefined', () => {
        const pendingInfo: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.PENDING,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.isBusyLoadingBookmarkInfo(pendingInfo)).to.equal(
          true,
        );
        const initialInfo: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.INITIAL,
            data: undefined,
            error: undefined,
          },
        };
        expect(bookmarkHelpers.isBusyLoadingBookmarkInfo(initialInfo)).to.equal(
          true,
        );
        const undefinedInfo: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: undefined,
        };
        expect(
          bookmarkHelpers.isBusyLoadingBookmarkInfo(undefinedInfo),
        ).to.equal(true);
      });

      it('should return true if currentLocation search is different from location search', () => {
        const info: AppBookmarkInfo = {currentLocation: {search: '?q=old'}};
        const location = {search: '?q=new'};
        expect(
          bookmarkHelpers.isBusyLoadingBookmarkInfo(info, location),
        ).to.equal(true);
      });

      it('should return false if userSavedSearchBookmarkTask is complete and locations match', () => {
        const info: AppBookmarkInfo = {
          userSavedSearchBookmarkTask: {
            status: TaskStatus.COMPLETE,
            data: undefined,
            error: undefined,
          },
          currentLocation: {search: '?q=test'},
        };
        const location = {search: '?q=test'};
        expect(
          bookmarkHelpers.isBusyLoadingBookmarkInfo(info, location),
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
  });
});
