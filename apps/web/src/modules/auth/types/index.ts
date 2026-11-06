/**
 * Carried in router state from the Google step to the workspace form —
 * never persisted: the credential is a short-lived Google ID token.
 */
export interface GoogleSignupState {
  credential: string;
  email: string;
  name: string;
  /** A staff invitation waits for this email (open the WO's link instead). */
  invited?: boolean;
}

/** POST /auth/google: a session, or what the person must do next. */
export type GoogleAuthOutcome =
  | { signupRequired: true; email: string; name: string; invited?: boolean }
  | { linkRequired: true; email: string; name: string };
