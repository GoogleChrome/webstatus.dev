import {LitElement, TemplateResult, html} from 'lit';

// ServiceElement is the class for services.
// Typically, they should not render anything.
export class ServiceElement extends LitElement {
  protected render(): TemplateResult {
    return html`<slot></slot>`;
  }
}
