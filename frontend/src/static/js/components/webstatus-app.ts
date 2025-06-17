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

import {provide} from '@lit/context';
import {type Router} from '@vaadin/router';
import {
  type CSSResultGroup,
  LitElement,
  type TemplateResult,
  css,
  html,
} from 'lit';
import {customElement, property, query} from 'lit/decorators.js';

import {type AppSettings} from '../../../common/app-settings.js';
import {routerContext} from '../contexts/router-context.js';
import {SHARED_STYLES} from '../css/shared-css.js';
import {initRouter} from '../utils/app-router.js';
import './webstatus-header.js';
import './webstatus-page.js';
import './webstatus-services-container.js';

@customElement('webstatus-app')
export class WebstatusApp extends LitElement {
  @query('webstatus-page')
  pageElement?: LitElement;

  @provide({context: routerContext})
  router?: Router;

  @property({type: Object})
  settings!: AppSettings;

  @property({type: String, attribute: 'gtag-id'})
  gtagID?: string;

  connectedCallback(): void {
    super.connectedCallback();

    if (this.gtagID) {
      window.dataLayer = window.dataLayer || [];
      window.gtag = function () {
        // eslint-disable-next-line prefer-rest-params
        window.dataLayer.push(arguments);
      };
      window.gtag('js', new Date());
      window.gtag('config', this.gtagID);
    } else {
      console.error(
        'gtag-id attribute is missing or empty. Google Analytics will not be initialized.',
      );
    }
  }

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
    if (this.pageElement !== null) {
      void initRouter(this.pageElement!).then((router: Router) => {
        this.router = router;
      });
    }
  }

  protected render(): TemplateResult {
    return html`
      <webstatus-services-container class="vbox" .settings="${this.settings}">
        <webstatus-header></webstatus-header>
        <webstatus-page class="halign-stretch valign-stretch">
          <slot></slot>
        </webstatus-page>
      </webstatus-services-container>
    `;
  }
}
