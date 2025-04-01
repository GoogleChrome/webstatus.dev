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

import {provide} from '@lit/context';
import {LitElement, TemplateResult, html} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import {
  appSettingsContext,
  AppSettings,
} from '../../contexts/settings-context.js';
import {WebstatusOverviewHeader} from '../webstatus-overview-header.js';
import '../webstatus-overview-header.js';
import {expect} from '@open-wc/testing';
import {
  AppBookmarkInfo,
  appBookmarkInfoContext,
} from '../../contexts/app-bookmark-info-context.js';

@customElement('fake-parent-element')
class FakeParentElement extends LitElement {
  @provide({context: appSettingsContext})
  @property({type: Object})
  settings!: AppSettings;

  @provide({context: appBookmarkInfoContext})
  @property({type: Object})
  appBookmarkInfo: AppBookmarkInfo = {};

  render(): TemplateResult {
    return html`<slot></slot>`;
  }
}

describe('webstatus-overview-header', () => {
  describe('RenderBookmarkUI', () => {
    let container: HTMLElement;
    afterEach(() => {
      document.body.removeChild(container);
    });

    it('should display the bookmark title and description when query is matched', async () => {
      container = document.createElement('div');
      container.innerHTML = `
        <fake-parent-element>
          <webstatus-overview-header>
          </webstatus-overview-header>
        </fake-parent-element>
      `;
      const parent: FakeParentElement = container.querySelector(
        'fake-parent-element',
      ) as FakeParentElement;
      const element: WebstatusOverviewHeader = container.querySelector(
        'webstatus-overview-header',
      ) as WebstatusOverviewHeader;
      // Set location to one of the globalBookmarks.
      element.location = {search: '?q=test_query_1'};
      parent.appBookmarkInfo = {
        globalBookmarks: [
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
        ],
        currentGlobalBookmark: {
          name: 'Test Bookmark 1',
          query: 'test_query_1',
          description: 'test description1',
        },
      };
      document.body.appendChild(container);
      await parent.updateComplete;
      await element.updateComplete;

      const title = element?.shadowRoot?.querySelector('#overview-title');
      expect(title).to.exist;
      expect(title!.textContent!.trim()).to.equal('Test Bookmark 1');

      const description = element?.shadowRoot?.querySelector(
        '#overview-description',
      );
      expect(description).to.exist;
      expect(description!.textContent).to.contain('test description1');
    });
    it('should not display description UI when it is empty', async () => {
      container = document.createElement('div');
      container.innerHTML = `
        <fake-parent-element>
          <webstatus-overview-header>
          </webstatus-overview-header>
        </fake-parent-element>
      `;
      const parent: FakeParentElement = container.querySelector(
        'fake-parent-element',
      ) as FakeParentElement;
      const element: WebstatusOverviewHeader = container.querySelector(
        'webstatus-overview-header',
      ) as WebstatusOverviewHeader;
      // Set location to one of the globalBookmarks.
      element.location = {search: '?q=test_query_1'};
      parent.appBookmarkInfo = {
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
      };
      document.body.appendChild(container);
      await parent.updateComplete;
      await element.updateComplete;

      const title = element?.shadowRoot?.querySelector('#overview-title');
      expect(title).to.exist;
      expect(title!.textContent!.trim()).to.equal('Test Bookmark 1');

      const description = element?.shadowRoot?.querySelector(
        '#overview-description',
      );
      expect(description).to.not.exist;
    });
  });
});
