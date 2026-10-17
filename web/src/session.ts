import { type Me, NotAllowed } from "./api";
import { callbackPath } from "./auth";

export type Session =
  | { kind: "loggedOut"; loginFailed?: boolean }
  | { kind: "loggedIn"; name: string }
  | { kind: "notAllowed" };

// The parts of oidc-client-ts's User and UserManager this module needs, so
// the tests don't need a browser.
export interface StoredUser {
  access_token: string;
  expired: boolean | undefined;
  refresh_token?: string;
}

export interface Login {
  signinRedirectCallback(): Promise<unknown>;
  getUser(): Promise<StoredUser | null>;
  signinSilent(): Promise<StoredUser | null>;
  removeUser(): Promise<void>;
}

export async function resume(
  login: Login,
  path: string,
  fetchMe: (accessToken: string) => Promise<Me | null>,
): Promise<Session> {
  if (path === callbackPath) {
    try {
      await login.signinRedirectCallback();
    } catch (error) {
      // a refused or abandoned login, or an old callback URL from the
      // history. The person can just try again.
      console.warn("finishing the login failed", error);

      return { kind: "loggedOut", loginFailed: true };
    }
  }

  let user = await login.getUser();
  if (user?.expired && user.refresh_token) {
    // a reload after the tokens expired would log the person out otherwise
    user = await login.signinSilent().catch(() => null);
  }

  if (!user || user.expired) {
    return { kind: "loggedOut" };
  }

  let me: Me | null;
  try {
    me = await fetchMe(user.access_token);
  } catch (error) {
    // the login stays, logging out of the issuer needs it
    if (error instanceof NotAllowed) {
      return { kind: "notAllowed" };
    }

    throw error;
  }

  if (me === null) {
    await login.removeUser();

    return { kind: "loggedOut" };
  }

  return { kind: "loggedIn", name: me.name };
}
