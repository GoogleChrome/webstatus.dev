/**
 * Copyright 2025 Google LLC
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

export function createMockIterator<T>(data: T[]) {
  return {
    [Symbol.asyncIterator]: () => ({
      next: async (): Promise<IteratorResult<T[]>> => {
        const value = data.shift();
        if (value) {
          return {
            value: [value],
            done: false,
          };
        } else {
          return {
            value: undefined,
            done: true,
          };
        }
      },
    }),
  };
}

// Can't use await el.updateComplete.
// Inspired from the lit/tasks tests themselves:
// https://github.com/lit/lit/blob/main/packages/task/src/test/task_test.ts
export const taskUpdateComplete = () =>
  new Promise(resolve => requestAnimationFrame(resolve));
