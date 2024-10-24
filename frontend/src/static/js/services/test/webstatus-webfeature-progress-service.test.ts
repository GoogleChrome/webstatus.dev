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

import {assert, expect, html} from '@open-wc/testing';
import {customElement, property} from 'lit/decorators.js';
import {consume, provide} from '@lit/context';
import {LitElement, TemplateResult} from 'lit';
import sinon from 'sinon';
import '../webstatus-webfeature-progress-service.js';
import {
  AppSettings,
  appSettingsContext,
} from '../../contexts/settings-context.js';
import {WebstatusWebFeatureProgressService} from '../webstatus-webfeature-progress-service.js';
import {
  WebFeatureProgress,
  webFeatureProgressContext,
} from '../../contexts/webfeature-progress-context.js';

@customElement('fake-parent-element')
class FakeParentElement extends LitElement {
  @provide({context: appSettingsContext})
  @property({type: Object})
  settings!: AppSettings;

  render(): TemplateResult {
    return html`<slot></slot>`;
  }
}

@customElement('fake-child-progress-element')
class FakeChildElement extends LitElement {
  @consume({context: webFeatureProgressContext, subscribe: true})
  @property({attribute: false})
  progress?: WebFeatureProgress;
}

describe('webstatus-webfeature-progress-service', () => {
  let fetchStub: sinon.SinonStub;
  let child: FakeChildElement;
  let parent: FakeParentElement;
  let element: WebstatusWebFeatureProgressService;
  let container: HTMLElement;
  beforeEach(async () => {
    fetchStub = sinon.stub(window, 'fetch');
    container = document.createElement('div');
    container.innerHTML = `
      <fake-parent-element>
        <webstatus-webfeature-progress-service>
          <fake-child-progress-element></fake-child-progress-element>
        </webstatus-webfeature-progress-service>
      </fake-parent-element>
    `;
    parent = container.querySelector(
      'fake-parent-element'
    ) as FakeParentElement;

    element = container.querySelector(
      'webstatus-webfeature-progress-service'
    ) as WebstatusWebFeatureProgressService;

    child = container.querySelector(
      'fake-child-progress-element'
    ) as FakeChildElement;
    document.body.appendChild(container);
    await parent.updateComplete;
    await element.updateComplete;
    await child.updateComplete;
  });
  afterEach(() => {
    document.body.removeChild(container);
    sinon.restore();
  });
  const appSettings = {
    apiUrl: 'http://localhost',
    webFeaturesProgressUrl: 'url',
    firebase: {
      app: {
        apiKey: 'testapikey',
        authDomain: 'testauthdomain',
      },
      auth: {
        emulatorURL: 'http://localhost:9099',
        tenantID: 'tenantID',
      },
    },
  };
  it('can be added to the page', async () => {
    assert.exists(element);
  });
  it('can receive the app settings via context', async () => {
    parent.settings = appSettings;
    await parent.updateComplete;
    await element.updateComplete;
    await child.updateComplete;
    assert.equal(parent.settings, appSettings);
    assert.equal(element.appSettings, parent.settings);
  });

  it('can retrieve the progress and provide it to child elements', async () => {
    const progressData = {
      bcd_map_progress: 63.5,
    };
    fetchStub.resolves(new Response(JSON.stringify(progressData)));
    await element.loadProgress('url');
    expect(fetchStub).to.have.been.calledOnceWith('url');
    const expectedProgress = {
      bcdMapProgress: 63.5,
    };
    expect(element.progress).to.deep.equal(expectedProgress);
    expect(child.progress).to.deep.equal(expectedProgress);
  });

  it('can retrieve the disabled progress and provide it to child elements', async () => {
    const progressData = {
      is_disabled: true,
    };
    fetchStub.resolves(new Response(JSON.stringify(progressData)));
    await element.loadProgress('url');
    expect(fetchStub).to.have.been.calledOnceWith('url');
    const expectedProgress = {
      isDisabled: true,
    };
    expect(element.progress).to.deep.equal(expectedProgress);
    expect(child.progress).to.deep.equal(expectedProgress);
  });

  it('can send a http error to child elements', async () => {
    fetchStub.resolves(new Response('', {status: 500}));
    await element.loadProgress('url');
    expect(fetchStub).to.have.been.calledOnceWith('url');
    const expectedProgress = {
      error: 'Received 500 status trying to get web feature stats',
    };
    expect(element.progress).to.deep.equal(expectedProgress);
    expect(child.progress).to.deep.equal(expectedProgress);
  });

  it('can send unknown error to child elements', async () => {
    fetchStub.rejects(new Error('Network error'));
    await element.loadProgress('url');
    expect(fetchStub).to.have.been.calledOnceWith('url');
    const expectedProgress = {
      error:
        'Unexpected error Error: Network error trying to get web feature stats',
    };
    expect(element.progress).to.deep.equal(expectedProgress);
    expect(child.progress).to.deep.equal(expectedProgress);
  });
});
