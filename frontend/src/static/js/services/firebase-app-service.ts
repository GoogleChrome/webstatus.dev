import {consume, provide} from '@lit/context';
import {PropertyValueMap} from 'lit';
import {customElement, state} from 'lit/decorators.js';
import {AppSettings, appSettingsContext} from '../contexts/settings-context.js';
import {
  FirebaseApp,
  firebaseAppContext,
} from '../contexts/firebase-app-context.js';
import {initializeApp} from 'firebase/app';
import {ServiceElement} from './service-element.js';

@customElement('firebase-app-service')
export class FirebaseAppService extends ServiceElement {
  @consume({context: appSettingsContext})
  @state()
  appSettings?: AppSettings;

  @provide({context: firebaseAppContext})
  @state()
  firebaseApp?: FirebaseApp;

  protected updated(
    _changedProperties: PropertyValueMap<any> | Map<PropertyKey, unknown>
  ): void {
    if (this.appSettings) {
      this.firebaseApp = initializeApp({
        projectId: 'local',
        apiKey: 'local',
        authDomain: 'local',
      });
    }
  }
}
