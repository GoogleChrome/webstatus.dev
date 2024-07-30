import {createContext} from '@lit/context';

import type {FirebaseApp} from 'firebase/app';
export type {FirebaseApp} from 'firebase/app';

export const firebaseAppContext = createContext<FirebaseApp | undefined>(
  'firebase-app'
);
