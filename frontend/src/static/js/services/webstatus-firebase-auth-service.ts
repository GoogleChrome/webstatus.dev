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

import {customElement, property, state} from 'lit/decorators.js';
import {consume, provide} from '@lit/context';
import {
  FirebaseApp,
  firebaseAppContext,
} from '../contexts/firebase-app-context.js';
import {
  AuthConfig,
  firebaseAuthContext,
} from '../contexts/firebase-auth-context.js';
import {
  Auth,
  GithubAuthProvider,
  connectAuthEmulator,
  getAuth,
  signInWithPopup,
} from 'firebase/auth';
import {User, firebaseUserContext} from '../contexts/firebase-user-context.js';
import {ServiceElement} from './service-element.js';

interface FirebaseAuthSettings {
  emulatorURL: string;
  tenantID: string;
}

@customElement('webstatus-firebase-auth-service')
export class WebstatusFirebaseAuthService extends ServiceElement {
  @property({type: Object})
  settings!: FirebaseAuthSettings;

  @consume({context: firebaseAppContext, subscribe: true})
  @state()
  firebaseApp?: FirebaseApp;

  @provide({context: firebaseAuthContext})
  firebaseAuthConfig?: AuthConfig;

  @provide({context: firebaseUserContext})
  user?: User;

  // Useful for testing
  authInitializer: (app: FirebaseApp | undefined) => Auth = getAuth;

  // Useful for testing
  emulatorConnector: (auth: Auth, url: string) => void = connectAuthEmulator;

  initFirebaseAuth() {
    if (this.firebaseApp) {
      const auth = this.authInitializer(this.firebaseApp);
      // Local environment will not have a tenantID.
      if (this.settings.tenantID !== '') {
        auth.tenantId = this.settings.tenantID;
      }
      const provider = new GithubAuthProvider();
      this.firebaseAuthConfig = {
        auth: auth,
        signIn: () => signInWithPopup(auth, provider),
        // Default to using the Github Provider
        provider: provider,
        icon: 'github',
      };
      if (this.settings.emulatorURL !== '') {
        this.emulatorConnector(
          this.firebaseAuthConfig.auth,
          this.settings.emulatorURL,
        );
      }
      // Set up the callback that will detect when:
      // 1. The user first logs in
      // 2. Resuming a session
      this.firebaseAuthConfig.auth.onAuthStateChanged(user => {
        this.user = user ? user : undefined;
      });
    }
  }

  protected firstUpdated(): void {
    this.initFirebaseAuth();
  }
}
