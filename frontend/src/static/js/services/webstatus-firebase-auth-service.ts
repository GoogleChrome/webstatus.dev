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
import {Task} from '@lit/task';
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
import {
  User,
  FirebaseUser,
  firebaseUserContext,
} from '../contexts/firebase-user-context.js';
import {ServiceElement} from './service-element.js';
interface FirebaseAuthSettings {
  emulatorURL: string;
  tenantID: string;
}

const GITHUB_USERNAME_REQUEST_URL: string = 'https://api.github.com/user/';

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
  user?: FirebaseUser;

  _loadingGithubUsername?: Task;

  // Useful for testing
  authInitializer: (app: FirebaseApp | undefined) => Auth = getAuth;

  // Useful for testing
  emulatorConnector: (auth: Auth, url: string) => void = connectAuthEmulator;

  handleGithubUsernameError(res: Response, email: string): void {
    const err = new Error();
    if (res.status === 403 && res.headers.get('X-RateLimit-Remaining') == '0') {
      const resetsAtMS = Number(`${res.headers.get('X-RateLimit-Reset')}000`);
      err.message = `Status: ${res.status}, Rate limit exceeded, try again in ${Math.ceil((resetsAtMS - Date.now()) / 60000)}m`;
    } else if (res.status === 404) {
      err.message = `Status: ${res.status}, Could not find user data for github: ${email}`;
    } else {
      // add other cases if you want to handle them
      err.message = `Unexpected status code: ${res.status}`;
    }
    throw err;
  }

  async getGithubUser(token: string): Promise<any> {
    return fetch(`${GITHUB_USERNAME_REQUEST_URL}`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
      },
    });
  }

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
          this.settings.emulatorURL
        );
      }
      // Set up the callback that will detect when:
      // 1. The user first logs in
      // 2. Resuming a session
      this.firebaseAuthConfig.auth.onAuthStateChanged((user: User | null) => {
        if (user !== null) {
          this.user =
            user !== null ? {...user, gitHubUsername: 'empty'} : undefined;
          this._loadingGithubUsername = new Task(this, {
            args: (): [any] => [this.user],
            task: async ([firebaseUser]: [any]) => {
              try {
                if (
                  firebaseUser &&
                  typeof firebaseUser !== 'string' &&
                  firebaseUser.accessToken
                ) {
                  const res: Response = await this.getGithubUser(
                    firebaseUser.accessToken
                  );
                  if (res.ok) {
                    const data = await res.json();
                    if (this.user !== undefined) {
                      this.user = {
                        ...this.user,
                        gitHubUsername: data.login,
                      };
                    }
                  } else {
                    this.handleGithubUsernameError(res, this.user?.email || '');
                  }
                } else {
                  throw new Error('Github username request failed.');
                }
              } catch (e: any) {
                console.error(e.message);
              }
            },
          });
          this._loadingGithubUsername.run();
        } else {
          throw new Error('Github username request failed.');
        }
      });
    }
  }

  protected firstUpdated(): void {
    this.initFirebaseAuth();
  }
}
