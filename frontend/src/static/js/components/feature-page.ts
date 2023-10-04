import { html, LitElement } from "lit";
import {customElement, property} from 'lit/decorators.js';
import { Client } from "../api/client.js";
import { components } from "webstatus.dev-backend";


@customElement('feature-page')
export class FeaturePage extends LitElement {
    id!: string;

    @property()
    feature?: components["schemas"]["Feature"] | undefined;

    @property()
    loading: boolean = true;
  
    async firstUpdated() {
      const client = new Client("http://localhost:8080");
      this.feature = await client.getFeature(this.id); 
      this.loading = false;
    }
  
    render() {
      if (this.loading)
        return html`Loading`
      else
        return html`
          <h1>Feature Page</h1>
          spec size: ${this.feature?.spec? this.feature.spec.length : 0}
          <br/>
          Specs:
        `;
    }
}