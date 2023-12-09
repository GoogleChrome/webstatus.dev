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