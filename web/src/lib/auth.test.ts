import { describe, expect, it } from "vitest";

import { loginSettings } from "./auth";

describe("loginSettings", () => {
  it("logs in at the issuer and comes back to frogly", () => {
    // arrange
    const config = { issuer: "http://localhost:8082/oidc/frogly", clientId: "frogly" };

    // act
    const settings = loginSettings(config, "http://localhost:5173");

    // assert
    expect(settings).toMatchObject({
      authority: "http://localhost:8082/oidc/frogly",
      client_id: "frogly",
      redirect_uri: "http://localhost:5173/callback",
      post_logout_redirect_uri: "http://localhost:5173/",
      response_type: "code",
      scope: "openid profile email",
      automaticSilentRenew: true,
    });
  });
});
