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
import {WebstatusOverviewContent} from '../webstatus-overview-content.js';
import '../webstatus-overview-content.js';
import {assert, expect} from '@open-wc/testing';
import {Toast} from '../../utils/toast.js';
import sinon from 'sinon';

@customElement('fake-parent-element')
class FakeParentElement extends LitElement {
  @provide({context: appSettingsContext})
  @property({type: Object})
  settings!: AppSettings;

  render(): TemplateResult {
    return html`<slot></slot>`;
  }
}

describe('webstatus-overview-content', () => {
  describe('renderMappingPercentage', () => {
    let parent: FakeParentElement;
    let element: WebstatusOverviewContent;
    let container: HTMLElement;
    let testContainer: HTMLElement;
    beforeEach(async () => {
      container = document.createElement('div');
      container.innerHTML = `
        <fake-parent-element>
          <webstatus-overview-content>
          </webstatus-overview-content>
        </fake-parent-element>
      `;
      parent = container.querySelector(
        'fake-parent-element'
      ) as FakeParentElement;

      element = container.querySelector(
        'webstatus-overview-content'
      ) as WebstatusOverviewContent;
      document.body.appendChild(container);
      await parent.updateComplete;
      await element.updateComplete;
      testContainer = element?.shadowRoot?.querySelector(
        '#mapping-percentage'
      ) as HTMLElement;
      assert.exists(testContainer);
    });
    afterEach(() => {
      document.body.removeChild(container);
    });
    it('should return an empty TemplateResult when webFeaturesProgress is undefined', () => {
      expect(testContainer.textContent?.trim()).to.equal('');
    });
    it('should return an empty TemplateResult when webFeaturesProgress is disabled', async () => {
      element.webFeaturesProgress = {isDisabled: true};
      await element.updateComplete;
      expect(testContainer.textContent?.trim()).to.equal('');
    });

    it('should call toast with the error message when webFeaturesProgress has an error', async () => {
      const toastStub = sinon.stub(Toast.prototype, 'toast');
      element.webFeaturesProgress = {error: 'Test error'};

      await element.updateComplete;
      expect(toastStub.calledOnce).to.be.true;
      expect(
        toastStub.calledWith('Test error', 'danger', 'exclamation-triangle')
      ).to.be.true;

      expect(testContainer.textContent?.trim()).to.equal('');
    });

    it('should render the mapping percentage when available', async () => {
      element.webFeaturesProgress = {bcdMapProgress: 75};
      await element.updateComplete;
      expect(testContainer.textContent?.trim()).to.match(
        /Percentage of features mapped:\s*75%/
      );
    });
  });
});
