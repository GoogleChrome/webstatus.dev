import { LitElement, TemplateResult, html, isServer } from "lit";
import { customElement, property } from "lit/decorators.js";
// import { provide } from '@lit/context'
import { apiClientContext } from '../contexts/api-client-context.js'
import { APIClient } from '../api/client.js'
import { ContextProvider } from "@lit/context";


@customElement('webstatus-app-settings')
export class WebstatusAppSettings extends LitElement {
  @property()
  apiURL?: string

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  apiClientProvider:any;

  connectedCallback(): void {
    super.connectedCallback()
    if(!isServer){
      console.log("adding prov")
      console.log(this.apiURL)
      if (this.apiURL) {
        this.apiClientProvider = new ContextProvider(this, { context: apiClientContext });
        this.apiClientProvider.setValue(new APIClient(this.apiURL))
        console.log("added prov")
      }
        
    } else {
      console.log("not adding")
    }
  }

  // @provide({ context: apiClientContext })
  //   apiClient = new APIClient("http://localhost:8080")

  protected render(): TemplateResult {
    return html`<slot></slot>`
  }
}