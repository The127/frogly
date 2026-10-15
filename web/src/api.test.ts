import { afterEach, describe, expect, it, vi } from "vitest";

import { fetchConfig, fetchMe } from "./api";

function respondWith(status: number, body: unknown): void {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify(body), { status })),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("fetchConfig", () => {
  it("gives the issuer and client ID", async () => {
    // arrange
    respondWith(200, { issuer: "http://localhost:8082/oidc/frogly", clientId: "frogly" });

    // act
    const config = await fetchConfig();

    // assert
    expect(config).toEqual({ issuer: "http://localhost:8082/oidc/frogly", clientId: "frogly" });
  });

  it("fails when the server can't answer", async () => {
    // arrange
    respondWith(503, { status: 503 });

    // act
    const config = fetchConfig();

    // assert
    await expect(config).rejects.toThrow("503");
  });
});

describe("fetchMe", () => {
  it("sends the access token and gives the name", async () => {
    // arrange
    respondWith(200, { name: "Karo" });

    // act
    const me = await fetchMe("an-access-token");

    // assert
    expect(me).toEqual({ name: "Karo" });
    expect(fetch).toHaveBeenCalledWith("/api/me", {
      headers: { Authorization: "Bearer an-access-token" },
    });
  });

  it("gives nothing when frogly refuses the token", async () => {
    // arrange
    respondWith(401, { status: 401 });

    // act
    const me = await fetchMe("an-expired-token");

    // assert
    expect(me).toBeNull();
  });

  it("fails when the server can't check the token", async () => {
    // arrange
    respondWith(503, { status: 503 });

    // act
    const me = fetchMe("a-fine-token");

    // assert
    await expect(me).rejects.toThrow("503");
  });
});
