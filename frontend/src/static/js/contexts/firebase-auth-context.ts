import {createContext} from '@lit/context';

import type {Auth} from 'firebase/auth';
export type {Auth} from 'firebase/auth';

export const firebaseAuthContext = createContext<Auth | undefined>('firebase-');
