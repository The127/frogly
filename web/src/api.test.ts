import { afterEach, describe, expect, it, vi } from "vitest";

import { createProject, fetchConfig, fetchMe, fetchProjects } from "./api";

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

describe("fetchProjects", () => {
  it("gives your projects", async () => {
    // arrange
    respondWith(200, [{ id: "p-1", name: "Garden" }]);

    // act
    const projects = await fetchProjects("an-access-token");

    // assert
    expect(projects).toEqual([{ id: "p-1", name: "Garden" }]);
    expect(fetch).toHaveBeenCalledWith("/api/projects", {
      headers: { Authorization: "Bearer an-access-token" },
    });
  });

  it("fails when frogly can't answer", async () => {
    // arrange
    respondWith(503, { status: 503 });

    // act
    const projects = fetchProjects("an-access-token");

    // assert
    await expect(projects).rejects.toThrow("503");
  });
});

describe("createProject", () => {
  it("sends the name and gives the new project", async () => {
    // arrange
    respondWith(201, { id: "p-1", name: "Garden" });

    // act
    const created = await createProject("an-access-token", "Garden");

    // assert
    expect(created).toEqual({ project: { id: "p-1", name: "Garden" } });
    expect(fetch).toHaveBeenCalledWith("/api/projects", {
      method: "POST",
      headers: { Authorization: "Bearer an-access-token", "Content-Type": "application/json" },
      body: JSON.stringify({ name: "Garden" }),
    });
  });

  it("gives frogly's reason when the name can't be used", async () => {
    // arrange
    respondWith(400, { status: 400, detail: "a project needs a name" });

    // act
    const created = await createProject("an-access-token", "  ");

    // assert
    expect(created).toEqual({ problem: "a project needs a name" });
  });

  it("fails when frogly can't answer", async () => {
    // arrange
    respondWith(500, { status: 500 });

    // act
    const created = createProject("an-access-token", "Garden");

    // assert
    await expect(created).rejects.toThrow("500");
  });
});
