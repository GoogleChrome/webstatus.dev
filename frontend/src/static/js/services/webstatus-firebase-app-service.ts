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
import {customElement, property, state} from 'lit/decorators.js';
import {
  FirebaseApp,
  firebaseAppContext,
} from '../contexts/firebase-app-context.js';
import {initializeApp} from 'firebase/app';
import {ServiceElement} from './service-element.js';

interface FirebaseSettings {
  apiKey: string;
  authDomain: string;
}

@customElement('webstatus-firebase-app-service')
export class WebstatusFirebaseAppService extends ServiceElement {
  @property({type: Object})
  settings?: FirebaseSettings;

  @provide({context: firebaseAppContext})
  @state()
  firebaseApp?: FirebaseApp;

  protected firstUpdated(): void {
    // Only initialize the app when the environment variables are not empty.
    //
    // TODO: Remove the below comment once Cloud Identity Platform is enabled in staging and production.
    // This will allow us to try this locally and eventually
    // enable this in staging and production.
    if (
      this.settings &&
      this.settings.apiKey !== '' &&
      this.settings.authDomain !== ''
    ) {
      this.firebaseApp = initializeApp({
        apiKey: this.settings.apiKey,
        authDomain: this.settings.authDomain,
      });
    }
  }
}
