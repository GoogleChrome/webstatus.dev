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

export const DRAWER_WIDTH_PX = 390;

// Determine if the browser looks like the user is on a mobile device.
// We assume that a small enough window width implies a mobile device.
export const NARROW_WINDOW_MAX_WIDTH = 700;

export const IS_MOBILE = (() => {
  // If innerWidth is non-zero, use it.
  // Otherwise, use the documentElement.clientWidth, if non-zero.
  // Otherwise, use the body.clientWidth.

  const width =
    window.innerWidth !== 0
      ? window.innerWidth
      : document.documentElement?.clientWidth !== 0
        ? document.documentElement.clientWidth
        : document.body.clientWidth;

  return width <= NARROW_WINDOW_MAX_WIDTH || width === 0;
})();
