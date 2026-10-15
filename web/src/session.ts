import type { Me } from "./api";

export type Session =
  | { kind: "loggedOut"; loginFailed?: boolean }
  | { kind: "loggedIn"; name: string };

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
  if (path === "/callback") {
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

  const me = await fetchMe(user.access_token);
  if (me === null) {
    await login.removeUser();

    return { kind: "loggedOut" };
  }

  return { kind: "loggedIn", name: me.name };
}
