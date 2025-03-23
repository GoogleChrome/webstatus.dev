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
import {Bookmark} from '../../utils/constants.js';
import {customElement, property} from 'lit/decorators.js';
import {provide} from '@lit/context';
import {LitElement, TemplateResult} from 'lit';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../../contexts/app-bookmark-info-context.js';

const testBookmarks: Bookmark[] = [
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

@customElement('fake-bookmark-parent-element')
class FakeBookmarkParentElement extends LitElement {
  @provide({context: appBookmarkInfoContext})
  @property({type: Object})
  appBookmarkInfo: AppBookmarkInfo = {
    globalBookmarks: testBookmarks,
    currentGlobalBookmark: undefined,
  };

  render(): TemplateResult {
    return html`<slot></slot>`;
  }
}

function createTestContainer(): HTMLElement {
  const container = document.createElement('div');
  container.innerHTML = `
  <fake-bookmark-parent-element>
    <webstatus-sidebar-menu>
    </webstatus-sidebar-menu>
  </fake-bookmark-parent-element>
`;
  return container;
}

describe('webstatus-sidebar-menu', () => {
  let el: WebstatusSidebarMenu;
  let parent: FakeBookmarkParentElement;
  let container: HTMLElement;

  beforeEach(async () => {
    container = createTestContainer();

    parent = container.querySelector<FakeBookmarkParentElement>(
      'fake-bookmark-parent-element',
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

    await el.updateComplete; // Wait for the component to update with the new bookmarks
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
  });

  it('updates the active bookmark query when the URL changes', async () => {
    // Set mock location to match a test bookmark
    (el.getLocation as sinon.SinonStub).returns({
      search: `?q=${el.appBookmarkInfo?.globalBookmarks?.[1].query}`,
      href: `http://localhost/?q=${el.appBookmarkInfo?.globalBookmarks?.[1].query}`,
    });
    parent.appBookmarkInfo = {
      globalBookmarks: testBookmarks,
      currentGlobalBookmark: testBookmarks[1],
    };

    el.updateActiveStatus();
    await parent.updateComplete;
    await el.updateComplete;
    expect(el.getLocation as sinon.SinonStub).to.be.called;
    expect(el.getActiveBookmarkQuery()).to.equal(
      el.appBookmarkInfo?.globalBookmarks?.[1].query,
    );
  });

  it('correctly handles bookmark clicks', async () => {
    // Get the whole tree item
    const bookmarkItem = el.shadowRoot?.querySelector(
      'sl-tree-item[id="bookmark0"]',
    ) as SlTreeItem;
    expect(bookmarkItem).to.exist;

    // Get the anchor element within the tree item
    const bookmarkAnchor = bookmarkItem.querySelector('a') as HTMLAnchorElement;
    expect(bookmarkAnchor).to.exist;

    expect(el.getActiveBookmarkQuery()).to.be.null;

    (el.getLocation as sinon.SinonStub).returns({
      search: `?q=${el.appBookmarkInfo?.globalBookmarks?.[0].query}`,
      href: `http://localhost/?q=${el.appBookmarkInfo?.globalBookmarks?.[0].query}`,
    });

    // Stub the click method to prevent default behavior
    const clickStub = sinon.stub(bookmarkAnchor, 'click');

    // Click the anchor
    bookmarkAnchor.click();
    parent.appBookmarkInfo = {
      globalBookmarks: testBookmarks,
      currentGlobalBookmark: testBookmarks[0],
    };
    await parent.updateComplete;
    await el.updateComplete;

    // Simulate popstate event.
    const popStateEvent = new PopStateEvent('popstate', {
      state: {},
    });
    window.dispatchEvent(popStateEvent);
    await el.updateComplete;

    // Assertions
    expect(clickStub.calledOnce).to.be.true;
    expect(el.getActiveBookmarkQuery()).to.equal(
      el.appBookmarkInfo?.globalBookmarks?.[0].query,
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

  it('marks the active bookmark item as selected on first load', async () => {
    // Mock the location to match the first bookmark BEFORE the fixture is created
    const getCurrentLocationStub = sinon.stub().returns({
      search: `?q=${testBookmarks[0].query}`,
      href: `http://localhost/?q=${testBookmarks[0].query}`,
    });
    const navigateToUrlStub = sinon.stub();

    // create the element after the stub is created
    el = await fixture<WebstatusSidebarMenu>(html`
      <webstatus-sidebar-menu
        .getLocation=${getCurrentLocationStub}
        .navigate=${navigateToUrlStub}
        .appBookmarkInfo=${{
          globalBookmarks: testBookmarks,
          currentGlobalBookmark: testBookmarks[0],
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
