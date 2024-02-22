#!/usr/bin/env node
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

import cpy from 'cpy';

const BROWSER_LOGO_DIR = '../node_modules/@browser-logos';
const IMG_DEST = '.postinstall/static/img';

await cpy(`${BROWSER_LOGO_DIR}/chrome-dev/*_24x24.png`, IMG_DEST);
await cpy(`${BROWSER_LOGO_DIR}/edge-dev/*_24x24.png`, IMG_DEST);
await cpy(`${BROWSER_LOGO_DIR}/firefox-nightly/*_24x24.png`, IMG_DEST);
await cpy(
  `${BROWSER_LOGO_DIR}/safari-technology-preview/*_24x24.png`,
  IMG_DEST,
  {rename: 'safari-preview_24x24.png'}
);

await cpy(`${BROWSER_LOGO_DIR}/chrome-dev/*_32x32.png`, IMG_DEST);
await cpy(`${BROWSER_LOGO_DIR}/edge-dev/*_32x32.png`, IMG_DEST);
await cpy(`${BROWSER_LOGO_DIR}/firefox-nightly/*_32x32.png`, IMG_DEST);
await cpy(
  `${BROWSER_LOGO_DIR}/safari-technology-preview/*_32x32.png`,
  IMG_DEST,
  {rename: 'safari-preview_32x32.png'}
);

console.log(`copied logos to ${IMG_DEST}`);
