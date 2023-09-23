import { html, LitElement } from "lit";
import {customElement} from 'lit/decorators.js';

@customElement('my-element')
export class NavigationBar extends LitElement {
  render() {
    return html`
      <ul>
        <li><a href="/">Home</a></li>
        <li><a href="/items">Items</a></li>
      </ul>
    `;
  }
}

