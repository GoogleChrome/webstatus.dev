import {LitElement, TemplateResult, html} from 'lit';
import {customElement, property} from 'lit/decorators.js';

@customElement('webstatus-sidebar-nav-tree-item')
export class WebstatusSidebarNavTreeItem extends LitElement {
  @property({type: String})
  id!: string;

  @property({type: Boolean})
  startExpanded!: boolean;

  @property({type: String})
  path!: string;

  // Must render to light DOM, so sl-tree works as intended.
  createRenderRoot() {
    return this;
  }

  render(): TemplateResult {
    return html` <sl-tree-item id="${this.id}" .expanded=${this.startExpanded}>
      <sl-icon name="menu-button"></sl-icon>
      <a class="${this.id}-link" href="${this.path}">Features</a>
      <slot></slot>
    </sl-tree-item>`;
  }
}
