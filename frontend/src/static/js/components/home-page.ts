import { html, LitElement } from "lit";
import {customElement, property} from 'lit/decorators.js';
import { Client } from "../api/client.js";
import { components } from "webstatus.dev-backend";


@customElement('home-page')
export class HomePage extends LitElement {
    @property()
    items: components["schemas"]["Feature"][] = [];
  
    async firstUpdated() {
      console.log("hi");
      const client = new Client("http://localhost:8080");
      this.items = await client.getFeatures();
  
    }
  
    render() {
      return html`
        <h1>Home Page</h1>
        items size: ${this.items.length}
        <br/>
        <ul>
          ${this.items.map(item => html`<li><a href="/items/${item.feature_id}">${item.feature_id}</a></li>`)}
        </ul>
      `;
    }
}