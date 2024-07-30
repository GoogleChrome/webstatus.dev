import {createContext} from '@lit/context';

import type {User} from 'firebase/auth';
export type {User} from 'firebase/auth';

export const firebaseUserContext = createContext<User | undefined>(
  'firebase-user'
);
