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

import Router from '@koa/router'
import Koa from 'koa'
import mount from 'koa-mount'
import staticFiles from 'koa-static'

import { type AppSettings } from '../common/app-settings.js'
import { renderBase } from './templates/_base.js'

export function loadAppSettingsFromProcess (): AppSettings {
  const apiUrl = process.env.API_URL
  if (apiUrl == null) {
    throw new Error('Missing API_URL env var')
  }
  const gsiClientId = process.env.GSI_CLIENT_ID
  if (gsiClientId == null) {
    throw new Error('Missing GSI_CLIENT_ID env var')
  }
  return {
    apiUrl,
    gsiClientId
  }
}

const app = new Koa()
const router = new Router()

const appSettings = loadAppSettingsFromProcess()

router.get('/', async (ctx) => {
  ctx.type = 'text/html'
  ctx.body = renderBase(
    appSettings,
    '<webstatus-overview-page></webstatus-overview-page>',
    '/public/js/overview.js')
})

app.use(router.routes())
app.use(mount('/public', staticFiles('dist/static')))
app.listen(5555)
