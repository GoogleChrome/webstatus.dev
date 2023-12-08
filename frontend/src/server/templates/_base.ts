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

import '../../static/js/components/webstatus-app.js'
import { type RenderResult, render } from '@lit-labs/ssr'
import { type TemplateResult, html } from 'lit'
export function * renderBase (page: TemplateResult): Generator<string | Promise<RenderResult>, void, undefined> {
  yield `
    <!DOCTYPE html>
    <html>
      <head>
        <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Inter">
        <link href="https://fonts.googleapis.com/icon?family=Material+Icons"
          rel="stylesheet">
        <!-- On browsers that don't yet support native declarative shadow DOM, a
            paint can occur after some or all pre-rendered HTML has been parsed,
            but before the declarative shadow DOM polyfill has taken effect. This
            paint is undesirable because it won't include any component shadow DOM.
            To prevent layout shifts that can result from this render, we use a
            "dsd-pending" attribute to ensure we only paint after we know
            shadow DOM is active. -->
        <style>
          body[dsd-pending] {
            display: none;
          }
          * {
            margin: 0;
            padding: 0;
          }
        </style>
        <!-- TODO: move reset to its own css file -->
        <title>Home Page</title>
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
      </head>
      <body dsd-pending>
        <script>
          if (HTMLTemplateElement.prototype.hasOwnProperty('shadowRoot')) {
            // This browser has native declarative shadow DOM support, so we can
            // allow painting immediately.
            document.body.removeAttribute('dsd-pending');
          }
        </script>
        <script src="/public/index.js"></script>
    `
  yield * render(html`
    <webstatus-app>
    ${page}
    </webstatus-app>
  `)
  yield `
        <script type="module">
        // Check if we require the template shadow root polyfill.
        if (!HTMLTemplateElement.prototype.hasOwnProperty('shadowRoot')) {
        // Fetch the template shadow root polyfill.
        const {hydrateShadowRoots} = await import(
            '/node_modules/@webcomponents/template-shadowroot/template-shadowroot.js'
        );

        // Apply the polyfill. This is a one-shot operation, so it is important
        // it happens after all HTML has been parsed.
        hydrateShadowRoots(document.body);

        // At this point, browsers without native declarative shadow DOM
        // support can paint the initial state of your components!
        document.body.removeAttribute('dsd-pending');
        }
    </script>

    </body>
    </html>
    `
}
