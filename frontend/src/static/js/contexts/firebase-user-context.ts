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

import {createContext} from '@lit/context';
import type {User as FirebaseUser} from 'firebase/auth';

export type SyncState = 'idle' | 'syncing' | 'error';

export interface User {
  user: FirebaseUser;
  syncState: SyncState;
}

// User means there is an authenticated user
// null means no authenticated user is active
// undefined means a decision has not been made yet about the current user.
export const firebaseUserContext = createContext<User | null | undefined>(
  'firebase-user',
);
