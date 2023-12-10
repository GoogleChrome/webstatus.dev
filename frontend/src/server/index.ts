/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import Koa from 'koa'
import Router from '@koa/router'
import mount from 'koa-mount'
import { renderBase } from './templates/_base.js'
import { html } from 'lit'
import { RenderResultReadable } from '@lit-labs/ssr/lib/render-result-readable.js'

import staticFiles from 'koa-static'

import { AppSettings } from '../common/app-settings.js'
export function loadAppSettingsFromProcess(): AppSettings{
  const apiUrl = process.env.API_URL!
  const gsiClientId = process.env.GSI_CLIENT_ID!
  // TODO: check if values are null or empty string
  return {
    apiUrl: apiUrl,
    gsiClientId: gsiClientId,
  }
}

const app = new Koa()
const router = new Router()

const appSettings = loadAppSettingsFromProcess()

router.get('/', async (ctx) => {
  ctx.type = 'text/html'
  ctx.body = new RenderResultReadable(renderBase(
    appSettings,
    html`<webstatus-overview-page></webstatus-overview-page>`))
})

app.use(router.routes())
app.use(mount('/public', staticFiles('static')))
app.listen(5555)
