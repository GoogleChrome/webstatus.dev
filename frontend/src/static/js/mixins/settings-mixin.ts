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

import {LitElement, isServer} from 'lit';
import { property } from 'lit/decorators.js';

export declare class SettingsMixinInterface {
  apiURL: string;
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/ban-types
type Constructor<T = {}> = new (...args: any[]) => T;

export const SettingsMixin = <T extends Constructor<LitElement>>(superClass: T) => {
  class SettingsMixinClass extends superClass {
    @property() apiURL = isServer ? process.env.API_URL : null;
  }
  // Cast return type to your mixin's interface intersected with the superClass type
  return SettingsMixinClass as Constructor<SettingsMixinInterface> & T;
}