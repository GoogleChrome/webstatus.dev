/**
 * Copyright 2024 Google LLC
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

import {TaskStatus} from '@lit/task';

/**
 * Represents the state and data associated with an asynchronous task.
 * Provides a convenient way to track the task's progress and access its results.
 * @template T The type of data expected from the completed task.
 */
export interface TaskTracker<T, E> {
  /** Status of the task */
  status: TaskStatus;

  /** Stores the error object if an error occurred, or undefined if no error. */
  error: E | Error | undefined;

  /** Stores the result data of the completed task, or undefined if not complete or in error state. */
  data: T | undefined;
}

/**
 * Represents an error that occurs when a task is not ready to execute.
 */
export class TaskNotReadyError extends Error {
  constructor() {
    super('Task not ready');
  }
}
