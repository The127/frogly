import type { UserManagerSettings } from "oidc-client-ts";

import type { Config } from "./api";

export function loginSettings(config: Config, origin: string): UserManagerSettings {
  return {
    authority: config.issuer,
    client_id: config.clientId,
    redirect_uri: `${origin}/callback`,
    post_logout_redirect_uri: `${origin}/`,
    response_type: "code",
    scope: "openid profile email",
    // renews the tokens with the refresh token before they expire, so a
    // person stays logged in while the tab is open
    automaticSilentRenew: true,
  };
}
