import { assert, fixture, html } from '@open-wc/testing'

import { type AppSettings } from '../../contexts/settings-context.js'
import { type WebstatusApp } from '../webstatus-app.js'

describe('webstatus-app', () => {
  it('can be added to the page with the settings', async () => {
    const settings: AppSettings = {
      apiUrl: 'http://localhost',
      gsiClientId: 'testclientid'
    }
    const component = await fixture<WebstatusApp>(
      html`
        <webstatus-app .settings=${settings}></webstatus-app>`)
    assert.exists(component)
    assert.equal(component.settings.apiUrl, 'http://localhost')
    assert.equal(component.settings.gsiClientId, 'testclientid')
  })
})
