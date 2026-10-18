import { describe, expect, it, vi } from "vitest";

import type { Me } from "./api";
import type { Result } from "./client";
import { type UserManager, type StoredUser, resume } from "./session";

function fakeLogin(stored: StoredUser | null, renewed: StoredUser | null = null): UserManager {
  return {
    signinRedirectCallback: vi.fn(async () => undefined),
    getUser: vi.fn(async () => stored),
    signinSilent: vi.fn(async () => renewed),
    removeUser: vi.fn(async () => undefined),
  };
}

const fresh: StoredUser = { access_token: "fresh-token", expired: false, refresh_token: "refresh" };
const stale: StoredUser = { access_token: "stale-token", expired: true, refresh_token: "refresh" };

function frogly(names: Record<string, string>): (token: string) => Promise<Result<Me>> {
  return async (token) => (token in names ? { ok: true, value: { name: names[token] } } : { ok: false, failure: { kind: "loggedOut" } });
}

describe("resume", () => {
  it("is logged out without a stored login", async () => {
    // arrange
    const login = fakeLogin(null);

    // act
    const session = await resume(login, "/", frogly({}));

    // assert
    expect(session).toEqual({ kind: "loggedOut" });
  });

  it("greets the person a stored login belongs to", async () => {
    // arrange
    const login = fakeLogin(fresh);

    // act
    const session = await resume(login, "/", frogly({ "fresh-token": "Karo" }));

    // assert
    expect(session).toEqual({ kind: "loggedIn", name: "Karo" });
  });

  it("finishes the login when Keyline sends the person back", async () => {
    // arrange
    const login = fakeLogin(fresh);

    // act
    await resume(login, "/callback", frogly({ "fresh-token": "Karo" }));

    // assert
    expect(login.signinRedirectCallback).toHaveBeenCalledOnce();
  });

  it("renews an expired login with its refresh token", async () => {
    // arrange
    const login = fakeLogin(stale, fresh);

    // act
    const session = await resume(login, "/", frogly({ "fresh-token": "Karo" }));

    // assert
    expect(login.signinSilent).toHaveBeenCalledOnce();
    expect(session).toEqual({ kind: "loggedIn", name: "Karo" });
  });

  it("is logged out when renewing fails", async () => {
    // arrange
    const login = fakeLogin(stale);
    login.signinSilent = vi.fn(async () => {
      throw new Error("the refresh token expired");
    });

    // act
    const session = await resume(login, "/", frogly({}));

    // assert
    expect(session).toEqual({ kind: "loggedOut" });
  });

  it("forgets a login frogly refuses", async () => {
    // arrange
    const login = fakeLogin(fresh);

    // act
    const session = await resume(login, "/", frogly({}));

    // assert
    expect(login.removeUser).toHaveBeenCalledOnce();
    expect(session).toEqual({ kind: "loggedOut" });
  });

  it("keeps a login that may not use frogly, so the person can log out", async () => {
    // arrange
    const login = fakeLogin(fresh);
    const fetchMe = async (): Promise<Result<Me>> => ({ ok: false, failure: { kind: "notAllowed" } });

    // act
    const session = await resume(login, "/", fetchMe);

    // assert
    expect(login.removeUser).not.toHaveBeenCalled();
    expect(session).toEqual({ kind: "notAllowed" });
  });

  it("fails when frogly can't say who is logged in", async () => {
    // arrange
    const login = fakeLogin(fresh);
    const fetchMe = async (): Promise<Result<Me>> => ({ ok: false, failure: { kind: "failed" } });

    // act
    const session = resume(login, "/", fetchMe);

    // assert
    await expect(session).rejects.toThrow("frogly can't say who is logged in");
    expect(login.removeUser).not.toHaveBeenCalled();
  });

  it("offers the login again when Keyline's answer can't be used", async () => {
    // arrange
    const login = fakeLogin(null);
    login.signinRedirectCallback = vi.fn(async () => {
      throw new Error("No matching state found in storage");
    });

    // act
    const session = await resume(login, "/callback", frogly({}));

    // assert
    expect(session).toEqual({ kind: "loggedOut", loginFailed: true });
  });
});
