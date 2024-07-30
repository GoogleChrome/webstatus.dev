import {PropertyValueMap} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {AppSettings, appSettingsContext} from '../contexts/settings-context.js';
import {consume, provide} from '@lit/context';
import {
  FirebaseApp,
  firebaseAppContext,
} from '../contexts/firebase-app-context.js';
import {Auth, firebaseAuthContext} from '../contexts/firebase-auth-context.js';
import {connectAuthEmulator, getAuth} from 'firebase/auth';
import {User, firebaseUserContext} from '../contexts/firebase-user-context.js';
import {ServiceElement} from './service-element.js';

@customElement('firebase-auth-service')
export class FirebaseAuthService extends ServiceElement {
  @consume({context: appSettingsContext, subscribe: true})
  @state()
  appSettings?: AppSettings;

  @consume({context: firebaseAppContext, subscribe: true})
  @state()
  firebaseApp?: FirebaseApp;

  @provide({context: firebaseAuthContext})
  firebaseAuth?: Auth;

  @provide({context: firebaseUserContext})
  user?: User;

  initialized: boolean = false;

  initFirebaseAuth() {
    if (!this.initialized && this.appSettings && this.firebaseApp) {
      this.firebaseAuth = getAuth(this.firebaseApp);
      if (this.appSettings.firebaseAuthEmulatorURL !== '') {
        connectAuthEmulator(
          this.firebaseAuth,
          this.appSettings.firebaseAuthEmulatorURL
        );
      }
      this.firebaseAuth.onAuthStateChanged(user => {
        this.user = user ? user : undefined;
      });
      this.initialized = true;
    }
  }

  protected updated(
    _changedProperties: PropertyValueMap<any> | Map<PropertyKey, unknown>
  ): void {
    this.initFirebaseAuth();
  }
}
