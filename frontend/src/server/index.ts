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

const app = new Koa()
const router = new Router()

router.get('/', async (ctx) => {
  ctx.type = 'text/html'
  ctx.body = new RenderResultReadable(renderBase(html`<webstatus-overview-page></webstatus-overview-page>`))
})

app.use(router.routes())
app.use(mount('/public', staticFiles('static')))
app.listen(5555)
