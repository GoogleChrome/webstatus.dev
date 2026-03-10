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

import {
  type CSSResultGroup,
  LitElement,
  type TemplateResult,
  css,
  html,
} from 'lit';
import {customElement, property, query} from 'lit/decorators.js';
import {type AppSettings} from '../../../common/app-settings.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {type Route} from '../utils/router-utils.js';

import './webstatus-overview-page.js';
import './webstatus-feature-page.js';
import './webstatus-stats-page.js';
import './webstatus-notfound-error-page.js';
import './webstatus-feature-gone-split-page.js';
import './webstatus-notification-channels-page.js';
import './webstatus-subscriptions-page.js';
import './webstatus-header.js';
import './webstatus-page.js';
import './webstatus-services-container.js';

@customElement('webstatus-app')
export class WebstatusApp extends LitElement {
  @query('webstatus-page')
  pageElement?: LitElement;

  @property({type: Object})
  settings!: AppSettings;

  private _routes: Route[] = [
    {
      component: 'webstatus-overview-page',
      path: '/',
    },
    {
      component: 'webstatus-feature-page',
      path: '/features/:featureId',
    },
    {
      component: 'webstatus-stats-page',
      path: '/stats',
    },
    {
      component: 'webstatus-notification-channels-page',
      path: '/settings/notification-channels',
    },
    {
      component: 'webstatus-subscriptions-page',
      path: '/settings/subscriptions',
    },
    {
      component: 'webstatus-feature-gone-split-page',
      path: '/errors-410/feature-gone-split',
    },
    {
      path: '*',
      component: 'webstatus-notfound-error-page',
    },
  ];

  static get styles(): CSSResultGroup {
    return [
      SHARED_STYLES,
      css`
        body {
          height: 100%;
          position: relative;
        }
        /* We have to specify the vbox style of the :host manually,
           because the vbox class is not available in index.html.
           Consequently, we have to specify the vbox item style of
           webstatus-services-container manually for the same reason. */
        :host {
          display: flex;
          flex-direction: column;
        }
        webstatus-services-container {
          flex-grow: 1;
        }
      `,
    ];
  }

  firstUpdated(): void {
    this.requestUpdate();
  }

  protected render(): TemplateResult {
    return html`
      <webstatus-services-container
        class="vbox"
        .settings="${this.settings}"
        .routes="${this._routes}"
        .renderHost="${this.pageElement}"
      >
        <webstatus-header></webstatus-header>
        <webstatus-page class="halign-stretch valign-stretch">
          <slot></slot>
        </webstatus-page>
      </webstatus-services-container>
    `;
  }
}
