/**
 * Copyright 2023 Google LLC
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

import {assert, fixture, html} from '@open-wc/testing';

import {type AppSettings} from '../../contexts/settings-context.js';
import {type WebstatusApp} from '../webstatus-app.js';

describe('webstatus-app', () => {
  it('can be added to the page with the settings', async () => {
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
    const component = await fixture<WebstatusApp>(
      html` <webstatus-app .settings=${settings}></webstatus-app>`
    );
    assert.exists(component);
    assert.equal(component.settings, settings);
  });
});
