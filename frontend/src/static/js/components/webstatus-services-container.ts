import {LitElement, TemplateResult, html} from 'lit';
import {customElement} from 'lit/decorators.js';

import '../services/firebase-app-service.js';
import '../services/firebase-auth-service.js';

@customElement('webstatus-services-container')
export class ServicesContainer extends LitElement {
  protected render(): TemplateResult {
    return html`
      <firebase-app-service>
        <firebase-auth-service><slot></slot></firebase-auth-service>
      </firebase-app-service>
    `;
  }
}
