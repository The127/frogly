import { afterEach, describe, expect, it, vi } from "vitest";

import { LoggedOut, type Result, both, call } from "./client";

function respondWith(status: number, body?: unknown): void {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(body === undefined ? null : JSON.stringify(body), { status })),
  );
}

const token = async () => "an-access-token";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("call", () => {
  it("sends the token and gives the answer", async () => {
    // arrange
    respondWith(200, { name: "Karo" });

    // act
    const result = await call<{ name: string }>("/api/me", { token });

    // assert
    expect(result).toEqual({ ok: true, value: { name: "Karo" } });
    expect(fetch).toHaveBeenCalledWith("/api/me", {
      method: "GET",
      headers: { Authorization: "Bearer an-access-token" },
    });
  });

  it("sends a body as JSON", async () => {
    // arrange
    respondWith(201, { id: "p-1" });

    // act
    await call("/api/projects", { token, method: "POST", body: { name: "Garden" } });

    // assert
    expect(fetch).toHaveBeenCalledWith("/api/projects", {
      method: "POST",
      headers: { Authorization: "Bearer an-access-token", "Content-Type": "application/json" },
      body: JSON.stringify({ name: "Garden" }),
    });
  });

  it("calls without a login when it has no token", async () => {
    // arrange
    respondWith(200, { issuer: "http://keyline" });

    // act
    await call("/api/config");

    // assert
    expect(fetch).toHaveBeenCalledWith("/api/config", { method: "GET", headers: {} });
  });

  it("gives nothing for an answer without a body", async () => {
    // arrange
    respondWith(204);

    // act
    const result = await call("/api/projects/p-1/leave", { token, method: "POST" });

    // assert
    expect(result).toEqual({ ok: true, value: undefined });
  });

  it.each([
    { status: 401, failure: { kind: "loggedOut" } },
    { status: 403, failure: { kind: "notAllowed" } },
    { status: 404, failure: { kind: "notFound" } },
    { status: 409, failure: { kind: "conflict" } },
    { status: 500, failure: { kind: "failed" } },
    { status: 503, failure: { kind: "failed" } },
  ])("turns a $status into $failure.kind", async ({ status, failure }) => {
    // arrange
    respondWith(status, { status });
    vi.spyOn(console, "error").mockImplementation(() => {});

    // act
    const result = await call("/api/projects", { token });

    // assert
    expect(result).toEqual({ ok: false, failure });
  });

  it("gives frogly's reason for a request it refuses", async () => {
    // arrange
    respondWith(400, { status: 400, detail: "a project needs a name" });

    // act
    const result = await call("/api/projects", { token, method: "POST", refused: "frogly can't create that project" });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "refused", problem: "a project needs a name" } });
  });

  it.each([
    { name: "a refusal without a reason", status: 400, body: { status: 400 } },
    { name: "a refusal that isn't JSON", status: 400, body: undefined },
  ])("gives the fallback reason for $name", async ({ status, body }) => {
    // arrange
    respondWith(status, body);

    // act
    const result = await call("/api/projects", { token, method: "POST", refused: "frogly can't create that project" });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "refused", problem: "frogly can't create that project" } });
  });

  it("gives its own reason for a body too large for frogly to read", async () => {
    // arrange
    respondWith(413, { status: 413 });

    // act
    const result = await call("/api/projects/p-1/ideas", {
      token,
      method: "POST",
      refused: "frogly can't use that idea",
      tooLarge: "an idea has at most 500 characters",
    });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "refused", problem: "an idea has at most 500 characters" } });
  });

  it("counts an answer frogly can't have sent as failed", async () => {
    // arrange
    vi.stubGlobal("fetch", vi.fn(async () => new Response("<html>a proxy's page</html>", { status: 200 })));
    vi.spyOn(console, "error").mockImplementation(() => {});

    // act
    const result = await call("/api/projects", { token });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "failed" } });
  });

  it("counts a refusal as failed when the call says nothing about refusals", async () => {
    // arrange
    respondWith(400, { status: 400, detail: "not a user ID" });
    vi.spyOn(console, "error").mockImplementation(() => {});

    // act
    const result = await call("/api/projects/p-1/members", { token, method: "POST" });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "failed" } });
  });

  it("counts a login that ended before the call as logged out", async () => {
    // arrange
    respondWith(200, {});

    // act
    const result = await call("/api/projects", {
      token: async () => {
        throw new LoggedOut();
      },
    });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "loggedOut" } });
    expect(fetch).not.toHaveBeenCalled();
  });

  it("counts a network that is down as failed", async () => {
    // arrange
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch");
      }),
    );
    vi.spyOn(console, "error").mockImplementation(() => {});

    // act
    const result = await call("/api/projects", { token });

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "failed" } });
  });
});

describe("both", () => {
  it("gives both values when both calls work", async () => {
    // arrange
    const a: Promise<Result<number>> = Promise.resolve({ ok: true, value: 1 });
    const b: Promise<Result<string>> = Promise.resolve({ ok: true, value: "two" });

    // act
    const result = await both(a, b);

    // assert
    expect(result).toEqual({ ok: true, value: [1, "two"] });
  });

  it.each([
    { name: "the first", first: { kind: "notFound" }, second: undefined, failure: { kind: "notFound" } },
    { name: "the second", first: undefined, second: { kind: "failed" }, failure: { kind: "failed" } },
    { name: "both, the first one's", first: { kind: "loggedOut" }, second: { kind: "failed" }, failure: { kind: "loggedOut" } },
  ])("gives the failure of $name", async ({ first, second, failure }) => {
    // arrange
    const a = Promise.resolve((first ? { ok: false, failure: first } : { ok: true, value: 1 }) as Result<number>);
    const b = Promise.resolve((second ? { ok: false, failure: second } : { ok: true, value: 2 }) as Result<number>);

    // act
    const result = await both(a, b);

    // assert
    expect(result).toEqual({ ok: false, failure });
  });
});
