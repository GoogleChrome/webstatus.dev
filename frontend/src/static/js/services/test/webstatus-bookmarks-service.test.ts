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
import {LitElement, html} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {consume} from '@lit/context';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../../contexts/app-bookmark-info-context.js';
import {WebstatusBookmarksService} from '../webstatus-bookmarks-service.js';
import {fixture, expect} from '@open-wc/testing';
import '../webstatus-bookmarks-service.js';
import {DEFAULT_BOOKMARKS} from '../../utils/constants.js';

@customElement('test-bookmark-consumer')
class TestBookmarkConsumer extends LitElement {
  @consume({context: appBookmarkInfoContext, subscribe: true})
  @state()
  appBookmarkInfo?: AppBookmarkInfo;

  render() {
    return html`<div>${this.appBookmarkInfo?.globalBookmarks?.length}</div>`;
  }
}

describe('webstatus-bookmarks-service', () => {
  it('can be added to the page with the defaults', async () => {
    const component = await fixture<WebstatusBookmarksService>(
      html`<webstatus-bookmarks-service> </webstatus-bookmarks-service>`,
    );
    expect(component).to.exist;
    const expectedInfo: AppBookmarkInfo = {
      globalBookmarks: DEFAULT_BOOKMARKS,
      currentGlobalBookmark: undefined,
    };
    expect(component!.appBookmarkInfo).to.deep.equal(expectedInfo);
  });
  it('provides appBookmarkInfo to consuming components', async () => {
    const el = await fixture<WebstatusBookmarksService>(html`
      <webstatus-bookmarks-service>
        <test-bookmark-consumer></test-bookmark-consumer>
      </webstatus-bookmarks-service>
    `);
    const expectedInfo: AppBookmarkInfo = {
      globalBookmarks: DEFAULT_BOOKMARKS,
      currentGlobalBookmark: undefined,
    };
    const consumer = el.querySelector<TestBookmarkConsumer>(
      'test-bookmark-consumer',
    );
    el.getLocation = () => {
      return {search: '', href: ''};
    };
    expect(el).to.exist;
    expect(consumer).to.exist;
    expect(el.appBookmarkInfo).to.deep.equal(expectedInfo);
  });

  it('updates appBookmarkInfo on popstate event', async () => {
    const el = await fixture<WebstatusBookmarksService>(html`
      <webstatus-bookmarks-service>
        <test-bookmark-consumer></test-bookmark-consumer>
      </webstatus-bookmarks-service>
    `);
    const consumer = el.querySelector<TestBookmarkConsumer>(
      'test-bookmark-consumer',
    );
    el._globalBookmarks = [
      {
        name: 'Test Bookmark 1',
        query: 'test_query_1',
      },
    ];
    el.appBookmarkInfo = {
      globalBookmarks: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalBookmark: undefined,
    };
    await el.updateComplete;
    await consumer!.updateComplete;

    // Initial state
    expect(consumer!.appBookmarkInfo).to.deep.equal({
      globalBookmarks: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalBookmark: undefined,
    });

    // Simulate popstate event with a query
    el.getLocation = () => {
      return {search: '?q=test_query_1', href: '?q=test_query_1'};
    };
    const popStateEvent = new PopStateEvent('popstate', {
      state: {},
    });
    window.dispatchEvent(popStateEvent);
    await el.updateComplete;
    await consumer!.updateComplete;

    // Updated state
    expect(consumer!.appBookmarkInfo).to.deep.equal({
      globalBookmarks: [
        {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
        },
      ],
      currentGlobalBookmark: {
        name: 'Test Bookmark 1',
        query: 'test_query_1',
      },
    });
  });
});
