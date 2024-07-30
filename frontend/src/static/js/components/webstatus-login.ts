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

/// <reference types="@types/google.accounts" />

import {consume} from '@lit/context';
import {LitElement, type TemplateResult, html} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';

import {
  type AppSettings,
  appSettingsContext,
} from '../contexts/settings-context.js';
import {TaskStatus} from '@lit/task';
import {Auth, GithubAuthProvider, User, signInWithPopup} from 'firebase/auth';
import {firebaseUserContext} from '../contexts/firebase-user-context.js';
import {firebaseAuthContext} from '../contexts/firebase-auth-context.js';

@customElement('webstatus-login')
export class WebstatusLogin extends LitElement {
  @consume({context: appSettingsContext})
  appSettings?: AppSettings;

  @property({type: Number})
  initializationStatus: TaskStatus = TaskStatus.INITIAL;

  @consume({context: firebaseAuthContext, subscribe: true})
  @state()
  firebaseAuth?: Auth;

  @consume({context: firebaseUserContext, subscribe: true})
  @state()
  user?: User;

  // firstUpdated(): void {
  //   // const app = initializeApp({
  //   //   projectId: 'local',
  //   //   apiKey: 'local',
  //   //   authDomain: 'local',
  //   // });
  //   // this.firebaseAuth = getAuth(app);
  //   // this.firebaseAuth.onAuthStateChanged(user => {
  //   //   // Resume the user if previously signed in.
  //   //   this.user = user ? user : undefined;
  //   // });
  //   // if (
  //   //   this.appSettings?.firebaseAuthEmulatorURL !== undefined &&
  //   //   this.appSettings?.firebaseAuthEmulatorURL !== ''
  //   // ) {
  //   //   connectAuthEmulator(
  //   //     this.firebaseAuth,
  //   //     this.appSettings?.firebaseAuthEmulatorURL
  //   //   );
  //   // }
  // }

  startSignInFlow() {
    if (this.firebaseAuth === undefined) {
      return;
    }
    signInWithPopup(this.firebaseAuth, new GithubAuthProvider())
      .then(result => {
        // This gives you a GitHub Access Token. You can use it to access the GitHub API.
        const credential = GithubAuthProvider.credentialFromResult(result);
        if (credential === null) {
          return;
        }
        console.log(credential.idToken);
        // const token = credential.accessToken;

        // The signed-in user info.
        // this.user = result.user;
        // this.user.getIdToken(true).then(value => {
        //   console.log(value);
        // });
        // console.log(this.user);
        // IdP data available using getAdditionalUserInfo(result)
        // ...
      })
      .catch(error => {
        // Handle Errors here.
        // const errorCode = error.code;
        // const errorMessage = error.message;
        // The email of the user's account used.
        // const email = error.customData.email;
        // The AuthCredential type that was used.
        // const credential = GithubAuthProvider.credentialFromError(error);
        // ...
        console.log(error);
      });
  }

  handleLogInClick() {
    console.log('log in click');
    if (this.user === undefined) {
      this.startSignInFlow();
      return;
    }
    this.user
      .getIdToken()
      .then(value => {
        console.log(value);
      })
      .catch(() => {
        this.startSignInFlow();
      });
  }

  handleLogOutClick() {
    if (this.firebaseAuth === undefined) {
      return;
    }

    this.firebaseAuth
      .signOut()
      .then(() => {
        console.log('signed out');
      })
      .catch(error => {
        console.error(error);
      });
  }

  renderLoginButton(): TemplateResult {
    return html`
      <sl-button variant="default" @click=${this.handleLogInClick}>
        <sl-icon slot="prefix" name="github"></sl-icon>
        Log in
      </sl-button>
    `;
  }

  renderAuthenticatedButton(user: User): TemplateResult {
    return html`
      <sl-dropdown>
        <sl-button slot="trigger" caret
          ><sl-icon slot="prefix" name="github"></sl-icon
          >${user.email}</sl-button
        >
        <sl-menu>
          <sl-menu-item @click=${this.handleLogOutClick}>Sign out</sl-menu-item>
        </sl-menu>
      </sl-dropdown>
    `;
  }

  render(): TemplateResult {
    if (this.user === undefined) {
      return this.renderLoginButton();
    }

    return this.renderAuthenticatedButton(this.user);
  }
}
