/**
 * Copyright 2024 Google LLC
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

import {expect, fixture, html} from '@open-wc/testing';
import sinon from 'sinon';
import {WebstatusSidebarMenu} from '../webstatus-sidebar-menu.js';
import {SlTreeItem} from '@shoelace-style/shoelace';
import '../webstatus-sidebar-menu.js';
import {GlobalSavedSearch, UserSavedSearch} from '../../utils/constants.js';
import {customElement, property} from 'lit/decorators.js';
import {provide} from '@lit/context';
import {LitElement, TemplateResult, render} from 'lit';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../../contexts/app-bookmark-info-context.js';
import {TaskStatus} from '@lit/task';
import {AppLocation} from '../../utils/app-router.js';

const testGlobalSavedSearches: GlobalSavedSearch[] = [
  {
    name: 'Test Bookmark 1',
    query: 'test_query_1',
    description: 'test description1',
  },
  {
    name: 'Test Bookmark 2',
    query: 'test_query_2',
    description: 'test description2',
  },
];

const testUserSavedSearches: UserSavedSearch[] = [
  {id: 'saved1', name: 'Saved 1', query: 'saved_query_1'},
  {id: 'saved2', name: 'Saved 2', query: 'saved_query_2'},
  {id: 'saved3', name: 'Saved 3', query: 'saved_query_3'},
];

@customElement('fake-saved-search-parent-element')
class FakeSavedSearchParentElement extends LitElement {
  @provide({context: appBookmarkInfoContext})
  @property({type: Object})
  appBookmarkInfo: AppBookmarkInfo = {
    globalSavedSearches: testGlobalSavedSearches,
    currentGlobalSavedSearch: undefined,
    userSavedSearchesTask: {
      status: TaskStatus.COMPLETE,
      data: testUserSavedSearches,
      error: undefined,
    },
    userSavedSearchTask: {
      status: TaskStatus.COMPLETE,
      data: undefined,
      error: undefined,
    },
  };

  render(): TemplateResult {
    return html`<slot></slot>`;
  }
}

function createTestContainer(): HTMLElement {
  const container = document.createElement('div');
  container.innerHTML = `
  <fake-saved-search-parent-element>
    <webstatus-sidebar-menu>
    </webstatus-sidebar-menu>
  </fake-saved-search-parent-element>
`;
  return container;
}

describe('webstatus-sidebar-menu', () => {
  let el: WebstatusSidebarMenu;
  let parent: FakeSavedSearchParentElement;
  let container: HTMLElement;

  beforeEach(async () => {
    container = createTestContainer();

    parent = container.querySelector<FakeSavedSearchParentElement>(
      'fake-saved-search-parent-element',
    )!;
    el = container.querySelector<WebstatusSidebarMenu>(
      'webstatus-sidebar-menu',
    )!;
    expect(parent).to.exist;
    expect(el).to.exist;

    // Mock router utility functions and initial location
    el.getLocation = sinon.stub().returns({
      search: '',
      href: 'http://localhost/',
    });
    el.navigate = sinon.stub();
    document.body.appendChild(container);

    await el.updateComplete; // Wait for the component to update with the new searches
  });

  afterEach(() => {
    sinon.restore();
    document.body.removeChild(container);
  });

  it('renders the correct structure with features and statistics sections', async () => {
    await expect(el).shadowDom.to.be.accessible();

    const tree = el.shadowRoot?.querySelector('sl-tree');
    const featuresItem = tree?.querySelector('#features-item');
    // const statsItem = tree?.querySelector('#statistics-item');
    const bookmarkItems = featuresItem?.querySelectorAll('sl-tree-item');
    const userBookmarksItem = tree?.querySelector('#your-bookmarks-list');

    expect(tree).to.exist;
    expect(featuresItem).to.exist;
    // expect(statsItem).to.exist;
    expect(bookmarkItems).to.have.lengthOf(2); // Number of test bookmarks

    // By default no bookmarks should be highlighted
    expect(bookmarkItems![0].selected).to.be.false;
    expect(bookmarkItems![1].selected).to.be.false;

    expect(bookmarkItems![0].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );
    expect(bookmarkItems![1].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );

    const userBookmarkItems =
      userBookmarksItem?.querySelectorAll('sl-tree-item');
    expect(userBookmarksItem).to.exist;
    expect(userBookmarkItems).to.have.lengthOf(3);
    expect(userBookmarkItems![0].selected).to.be.false;
    expect(userBookmarkItems![1].selected).to.be.false;
    expect(userBookmarkItems![2].selected).to.be.false;

    expect(userBookmarkItems![0].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );
    expect(userBookmarkItems![1].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );
    expect(userBookmarkItems![2].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );
  });

  it('renders user saved searches correctly', async () => {
    const userSavedBookmarksItems = el.shadowRoot?.querySelectorAll(
      'sl-tree-item[id^="userbookmark"]',
    );
    expect(userSavedBookmarksItems).to.have.lengthOf(
      testUserSavedSearches.length,
    );
    testUserSavedSearches.forEach((bookmark, index) => {
      const item = userSavedBookmarksItems![index] as SlTreeItem;
      expect(item.id).to.equal(`userbookmark${index}`);
      expect(item.textContent).to.contain(bookmark.name);
    });
  });

  it('updates the active bookmark query when the URL changes', async () => {
    // Set mock location to match a test bookmark
    const mockLocation = {
      search: `?q=${el.appBookmarkInfo?.globalSavedSearches?.[1].query}`,
      href: `http://localhost/?q=${el.appBookmarkInfo?.globalSavedSearches?.[1].query}`,
    };
    (el.getLocation as sinon.SinonStub).returns(mockLocation);
    parent.appBookmarkInfo = {
      globalSavedSearches: testGlobalSavedSearches,
      currentGlobalSavedSearch: testGlobalSavedSearches[1],
      currentLocation: mockLocation,
    };

    el.updateActiveStatus();
    await parent.updateComplete;
    await el.updateComplete;
    expect(el.getLocation as sinon.SinonStub).to.be.called;
    expect(el.activeQuery).to.equal(
      el.appBookmarkInfo?.globalSavedSearches?.[1].query,
    );
  });

  it('renders global bookmarks correctly without IDs', async () => {
    const bookmarkItems = el.shadowRoot?.querySelectorAll(
      'sl-tree-item[id^="globalbookmark"]',
    );
    expect(bookmarkItems).to.have.lengthOf(testGlobalSavedSearches.length);
    testGlobalSavedSearches.forEach((bookmark, index) => {
      const item = bookmarkItems![index] as SlTreeItem;
      expect(item.id).to.equal(`globalbookmark${index}`);
      expect(item.textContent).to.contain(bookmark.name);
    });
  });

  it('renders user saved searches correctly with IDs', async () => {
    const userSavedBookmarksItems = el.shadowRoot?.querySelectorAll(
      'sl-tree-item[id^="userbookmark"]',
    );
    expect(userSavedBookmarksItems).to.have.lengthOf(
      testUserSavedSearches.length,
    );
    testUserSavedSearches.forEach((bookmark, index) => {
      const item = userSavedBookmarksItems![index] as SlTreeItem;
      expect(item.id).to.equal(`userbookmark${index}`);
      expect(item.textContent).to.contain(bookmark.name);
    });
  });

  it('correctly handles bookmark clicks correctly for global bookmarks', async () => {
    // Get the whole tree item
    const bookmarkItem = el.shadowRoot?.querySelector(
      'sl-tree-item[id="globalbookmark0"]',
    ) as SlTreeItem;
    expect(bookmarkItem).to.exist;

    // Get the anchor element within the tree item
    const bookmarkAnchor = bookmarkItem.querySelector('a') as HTMLAnchorElement;
    expect(bookmarkAnchor).to.exist;

    expect(el.activeQuery).to.be.null;

    const mockLocation = {
      search: `?q=${el.appBookmarkInfo?.globalSavedSearches?.[0].query}`,
      href: `http://localhost/?q=${el.appBookmarkInfo?.globalSavedSearches?.[0].query}`,
    };
    (el.getLocation as sinon.SinonStub).returns(mockLocation);

    // Stub the click method to prevent default behavior
    const clickStub = sinon.stub(bookmarkAnchor, 'click');

    // Click the anchor. The parent element handles updating the currentGlobalSavedSearch
    bookmarkAnchor.click();
    parent.appBookmarkInfo = {
      globalSavedSearches: testGlobalSavedSearches,
      currentGlobalSavedSearch: testGlobalSavedSearches[0],
      currentLocation: mockLocation,
    };
    await parent.updateComplete;
    await el.updateComplete;

    // Assertions
    expect(clickStub.calledOnce).to.be.true;
    expect(el.activeQuery).to.equal(
      el.appBookmarkInfo?.globalSavedSearches?.[0].query,
    );

    const bookmarkItems = el.shadowRoot
      ?.querySelector('sl-tree')
      ?.querySelector('#features-item')!
      .querySelectorAll('sl-tree-item') as NodeListOf<SlTreeItem>;
    // Check that only the first bookmark is selected
    expect(bookmarkItems[0].selected).to.be.true;
    expect(bookmarkItems[1].selected).to.be.false;

    // Check the icon name based on the selected state
    expect(bookmarkItems[0].querySelector('sl-icon')?.name).to.equal(
      'bookmark-star',
    );
    expect(bookmarkItems[1].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );

    // Restore
    clickStub.restore();
  });

  it('correctly handles bookmark clicks for user saved searches, verifying isQueryActive', async () => {
    // Set up the URL to match the first user bookmark
    const mockLocation: AppLocation = {
      href: `http://localhost/?search_id=${testUserSavedSearches[0].id}`,
      search: `?search_id=${testUserSavedSearches[0].id}`,
      pathname: '/',
    };
    (el.getLocation as sinon.SinonStub).returns(mockLocation);
    parent.appBookmarkInfo = {
      ...parent.appBookmarkInfo,
      currentLocation: mockLocation,
    };
    el.updateActiveStatus();
    await el.updateComplete;

    const bookmarkItem = el.shadowRoot?.querySelector(
      'sl-tree-item[id="userbookmark0"]',
    ) as SlTreeItem;
    expect(bookmarkItem).to.exist;
    const bookmarkAnchor = bookmarkItem.querySelector('a') as HTMLAnchorElement;
    expect(bookmarkAnchor).to.exist;

    const clickStub = sinon.stub(bookmarkAnchor, 'click');
    bookmarkAnchor.click();

    await el.updateComplete; // Allow the component to update after the click

    expect(clickStub.calledOnce).to.be.true;
    expect(el.activeQuery).to.equal(testUserSavedSearches[0].query);

    // Assertions to check the selected state of other bookmarks
    const userBookmarkItems = el.shadowRoot?.querySelectorAll(
      'sl-tree-item[id^="userbookmark"]',
    ) as NodeListOf<SlTreeItem>;
    expect(userBookmarkItems[0].selected).to.be.true;
    expect(userBookmarkItems[1].selected).to.be.false;
    expect(userBookmarkItems[2].selected).to.be.false;

    // Check the icon name based on the selected state
    expect(userBookmarkItems[0].querySelector('sl-icon')?.name).to.equal(
      'bookmark-star',
    );
    expect(userBookmarkItems[1].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );
    expect(userBookmarkItems[2].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );

    clickStub.restore();
  });

  it('renders user saved searches section correctly - pending state', async () => {
    const userBookmarksSection = el.shadowRoot?.querySelector(
      '#your-bookmarks-list',
    );
    expect(userBookmarksSection).to.exist;
    expect(userBookmarksSection?.textContent).to.contain('Your Bookmarks');

    parent.appBookmarkInfo = {
      ...parent.appBookmarkInfo,
      userSavedSearchesTask: {
        status: TaskStatus.PENDING,
        data: undefined,
        error: undefined,
      },
    };
    await el.updateComplete;
    const userBookmarkItems = userBookmarksSection?.querySelectorAll(
      'sl-tree-item sl-skeleton',
    );
    expect(userBookmarkItems).to.have.lengthOf(3); // Should show 3 skeletons while pending
  });

  it('renders user saved searches section correctly - complete with data', async () => {
    const userBookmarksSection = el.shadowRoot?.querySelector(
      '#your-bookmarks-list',
    );
    expect(userBookmarksSection).to.exist;
    expect(userBookmarksSection?.textContent).to.contain('Your Bookmarks');

    parent.appBookmarkInfo = {
      ...parent.appBookmarkInfo,
      userSavedSearchesTask: {
        status: TaskStatus.COMPLETE,
        data: testUserSavedSearches,
        error: undefined,
      },
    };
    await el.updateComplete;
    const userBookmarkItems =
      userBookmarksSection?.querySelectorAll('sl-tree-item');
    expect(userBookmarkItems).to.have.lengthOf(testUserSavedSearches.length);
  });

  it('renders user saved searches section correctly - complete with empty data', async () => {
    const userBookmarksSection = el.shadowRoot?.querySelector(
      '#your-bookmarks-list',
    );
    expect(userBookmarksSection).to.exist;
    expect(userBookmarksSection?.textContent).to.contain('Your Bookmarks');

    parent.appBookmarkInfo = {
      ...parent.appBookmarkInfo,
      userSavedSearchesTask: {
        status: TaskStatus.COMPLETE,
        data: [],
        error: undefined,
      },
    };
    await el.updateComplete;
    const userBookmarkItems =
      userBookmarksSection?.querySelectorAll('sl-tree-item');
    expect(userBookmarkItems).to.have.lengthOf(0);
  });

  it('renders user saved searches section correctly - error state', async () => {
    const userBookmarksSection = el.shadowRoot?.querySelector(
      '#your-bookmarks-list',
    );
    expect(userBookmarksSection).to.exist;
    expect(userBookmarksSection?.textContent).to.contain('Your Bookmarks');

    parent.appBookmarkInfo = {
      ...parent.appBookmarkInfo,
      userSavedSearchesTask: {
        status: TaskStatus.ERROR,
        data: undefined,
        error: new Error('Failed to load bookmarks'),
      },
    };
    await el.updateComplete;
    const userBookmarkItems =
      userBookmarksSection?.querySelectorAll('sl-tree-item');
    expect(userBookmarkItems).to.have.lengthOf(0); // Should show nothing in rejected state
  });

  it('renders renderUserSavedSearch correctly with user search ID without ownership', async () => {
    const search = testUserSavedSearches[0];
    const renderedSearch = el.renderUserSavedSearch(search, 0);
    const container = document.createElement('div');
    render(renderedSearch, container);
    expect(container.querySelector('sl-tree-item')?.id).to.equal(
      'userbookmark0',
    );
    expect(container.textContent).to.contain(search.name);
    expect(
      container
        .querySelector('sl-tree-item')
        ?.querySelector('.saved-search-edit-link'),
    ).to.be.null;
  });

  it('renders renderUserSavedSearch correctly with user search ID with ownership', async () => {
    const search = testUserSavedSearches[0];
    search.permissions = {role: 'saved_search_owner'};
    const renderedSearch = el.renderUserSavedSearch(search, 0);
    const container = document.createElement('div');
    render(renderedSearch, container);
    expect(container.querySelector('sl-tree-item')?.id).to.equal(
      'userbookmark0',
    );
    expect(container.textContent).to.contain(search.name);
    expect(
      container
        .querySelector('sl-tree-item')
        ?.querySelector('.saved-search-edit-link'),
    ).exist;
  });

  it('renders renderGlobalSavedSearch correctly without ID for global saved search', async () => {
    const bookmark = testGlobalSavedSearches[0];
    const renderedSearch = el.renderGlobalSavedSearch(bookmark, 0);
    const container = document.createElement('div');
    render(renderedSearch, container);
    expect(container.querySelector('sl-tree-item')?.id).to.equal(
      'globalbookmark0',
    );
    expect(container.textContent).to.contain(bookmark.name);
  });

  it('marks the active search item as selected on first load', async () => {
    // Mock the location to match the first search BEFORE the fixture is created
    const mockLocation = {
      search: `?q=${testGlobalSavedSearches[0].query}`,
      href: `http://localhost/?q=${testGlobalSavedSearches[0].query}`,
    };
    const getCurrentLocationStub = sinon.stub().returns(mockLocation);
    const navigateToUrlStub = sinon.stub();

    // create the element after the stub is created
    el = await fixture<WebstatusSidebarMenu>(html`
      <webstatus-sidebar-menu
        .getLocation=${getCurrentLocationStub}
        .navigate=${navigateToUrlStub}
        .appBookmarkInfo=${{
          globalSavedSearches: testGlobalSavedSearches,
          currentGlobalSavedSearch: testGlobalSavedSearches[0],
          currentLocation: mockLocation,
        }}
      ></webstatus-sidebar-menu>
    `);

    await el.updateComplete;

    const bookmarkItems = el.shadowRoot
      ?.querySelector('sl-tree')
      ?.querySelector('#features-item')!
      .querySelectorAll('sl-tree-item') as NodeListOf<SlTreeItem>;

    // Check that only the first bookmark is selected
    expect(bookmarkItems[0].selected).to.be.true;
    expect(bookmarkItems[1].selected).to.be.false;

    // Check the icon name based on the selected state
    expect(bookmarkItems[0].querySelector('sl-icon')?.name).to.equal(
      'bookmark-star',
    );
    expect(bookmarkItems[1].querySelector('sl-icon')?.name).to.equal(
      'bookmark',
    );
  });
});
