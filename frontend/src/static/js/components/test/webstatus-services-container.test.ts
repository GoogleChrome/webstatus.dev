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

import {fixture, assert} from '@open-wc/testing';
import {html} from 'lit';
import {WebstatusServicesContainer} from './../webstatus-services-container.js';
import {AppSettings} from '../../../../common/app-settings.js';
import './../webstatus-services-container.js';
import {ServiceElement} from '../../services/service-element.js';

describe('WebstatusServiceContainer', () => {
  const settings: AppSettings = {
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

  const slotContent = html`<div id="slot-content">
    This is the slot content
  </div>`;

  async function initializeServices(el: WebstatusServicesContainer) {
    await el.updateComplete; // Wait for initial update

    const serviceTags = [
      'webstatus-gcharts-loader-service',
      'webstatus-app-settings-service',
      'webstatus-api-client-service',
      'webstatus-firebase-app-service',
      'webstatus-firebase-auth-service',
    ];

    for (const serviceTag of serviceTags) {
      const service = el.shadowRoot?.querySelector(
        serviceTag,
      ) as ServiceElement;
      try {
        assert.exists(service, `${serviceTag} should be rendered`);
        await service.updateComplete;
      } catch (error) {
        // Handle potential errors during service initialization
        console.error(`Error initializing ${serviceTag}:`, error);
        assert.fail(`Error initializing ${serviceTag}`);
      }
    }
  }

  it('should render all services correctly', async () => {
    const el = await fixture<WebstatusServicesContainer>(html`
      <webstatus-services-container
        .settings=${settings}
      ></webstatus-services-container>
    `);

    await initializeServices(el);
  });

  it('should render slot content correctly', async () => {
    const el = await fixture<WebstatusServicesContainer>(html`
      <webstatus-services-container .settings=${settings}>
        ${slotContent}
      </webstatus-services-container>
    `);

    await initializeServices(el);

    const slotContentElement = el.querySelector('#slot-content');
    assert.exists(slotContentElement, 'Slot content element should exist');
    assert.strictEqual(
      slotContentElement.textContent?.trim(),
      'This is the slot content',
      'Slot content should match',
    );
  });
});
