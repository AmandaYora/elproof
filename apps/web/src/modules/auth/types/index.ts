/**
 * Carried in router state from the Google step to the workspace form —
 * never persisted: the credential is a short-lived Google ID token.
 */
export interface GoogleSignupState {
  credential: string;
  email: string;
  name: string;
}

/** POST /auth/google: a session, or what the person must do next. */
export type GoogleAuthOutcome =
  | { signupRequired: true; email: string; name: string }
  | { linkRequired: true; email: string; name: string };
