import { afterEach, describe, expect, it, vi } from "vitest";

import { LoggedOut, NotFound, createProject, fetchConfig, fetchGarden, fetchMe, fetchProject, fetchProjects, plantIdea } from "./api";

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

describe("fetchProject", () => {
  it("gives the project", async () => {
    // arrange
    respondWith(200, { id: "p-1", name: "Garden" });

    // act
    const project = await fetchProject("an-access-token", "p-1");

    // assert
    expect(project).toEqual({ id: "p-1", name: "Garden" });
    expect(fetch).toHaveBeenCalledWith("/api/projects/p-1", {
      headers: { Authorization: "Bearer an-access-token" },
    });
  });

  it("throws NotFound for a project you can't see", async () => {
    // arrange
    respondWith(404, { status: 404 });

    // act
    const project = fetchProject("an-access-token", "p-1");

    // assert
    await expect(project).rejects.toBeInstanceOf(NotFound);
  });

  it("fails when frogly can't answer", async () => {
    // arrange
    respondWith(500, { status: 500 });

    // act
    const project = fetchProject("an-access-token", "p-1");

    // assert
    await expect(project).rejects.toThrow("500");
  });

  it("puts the project's ID into the path safely", async () => {
    // arrange
    respondWith(200, { id: "p-1", name: "Garden" });

    // act
    await fetchProject("an-access-token", "../me");

    // assert
    expect(fetch).toHaveBeenCalledWith("/api/projects/..%2Fme", expect.anything());
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

describe("an ended login", () => {
  it("makes fetchProjects throw LoggedOut", async () => {
    // arrange
    respondWith(401, { status: 401 });

    // act
    const projects = fetchProjects("an-expired-token");

    // assert
    await expect(projects).rejects.toBeInstanceOf(LoggedOut);
  });

  it("makes createProject throw LoggedOut", async () => {
    // arrange
    respondWith(401, { status: 401 });

    // act
    const created = createProject("an-expired-token", "Garden");

    // assert
    await expect(created).rejects.toBeInstanceOf(LoggedOut);
  });
});

describe("a refusal without a reason", () => {
  it("still gives a reason", async () => {
    // arrange
    vi.stubGlobal("fetch", vi.fn(async () => new Response("not JSON", { status: 400 })));

    // act
    const created = await createProject("an-access-token", "Garden");

    // assert
    expect(created).toEqual({ problem: "frogly can't use that name" });
  });
});

const idea = {
  id: "i-1",
  text: "search by any word",
  plantedBy: "Karo",
  plantedAt: "2026-10-15T21:00:00.123456Z",
};

describe("fetchGarden", () => {
  it("gives the project's ideas", async () => {
    // arrange
    respondWith(200, [idea]);

    // act
    const ideas = await fetchGarden("an-access-token", "p-1");

    // assert
    expect(ideas).toEqual([idea]);
    expect(fetch).toHaveBeenCalledWith("/api/projects/p-1/ideas", {
      headers: { Authorization: "Bearer an-access-token" },
    });
  });

  it("throws NotFound for a project you can't see", async () => {
    // arrange
    respondWith(404, { status: 404 });

    // act
    const ideas = fetchGarden("an-access-token", "p-1");

    // assert
    await expect(ideas).rejects.toBeInstanceOf(NotFound);
  });

  it("fails when frogly can't answer", async () => {
    // arrange
    respondWith(500, { status: 500 });

    // act
    const ideas = fetchGarden("an-access-token", "p-1");

    // assert
    await expect(ideas).rejects.toThrow("500");
  });

  it("throws LoggedOut when the login ended", async () => {
    // arrange
    respondWith(401, { status: 401 });

    // act
    const ideas = fetchGarden("an-expired-token", "p-1");

    // assert
    await expect(ideas).rejects.toBeInstanceOf(LoggedOut);
  });

  it("puts the project's ID into the path safely", async () => {
    // arrange
    respondWith(200, []);

    // act
    await fetchGarden("an-access-token", "../me");

    // assert
    expect(fetch).toHaveBeenCalledWith("/api/projects/..%2Fme/ideas", expect.anything());
  });
});

describe("plantIdea", () => {
  it("sends the text and gives the new idea", async () => {
    // arrange
    respondWith(201, idea);

    // act
    const planted = await plantIdea("an-access-token", "p-1", "search by any word");

    // assert
    expect(planted).toEqual({ idea });
    expect(fetch).toHaveBeenCalledWith("/api/projects/p-1/ideas", {
      method: "POST",
      headers: { Authorization: "Bearer an-access-token", "Content-Type": "application/json" },
      body: JSON.stringify({ text: "search by any word" }),
    });
  });

  it("gives frogly's reason when the text can't be used", async () => {
    // arrange
    respondWith(400, { status: 400, detail: "an idea needs a text" });

    // act
    const planted = await plantIdea("an-access-token", "p-1", "  ");

    // assert
    expect(planted).toEqual({ problem: "an idea needs a text" });
  });

  it("throws NotFound for a project you can't see", async () => {
    // arrange
    respondWith(404, { status: 404 });

    // act
    const planted = plantIdea("an-access-token", "p-1", "an idea");

    // assert
    await expect(planted).rejects.toBeInstanceOf(NotFound);
  });

  it("throws LoggedOut when the login ended", async () => {
    // arrange
    respondWith(401, { status: 401 });

    // act
    const planted = plantIdea("an-expired-token", "p-1", "an idea");

    // assert
    await expect(planted).rejects.toBeInstanceOf(LoggedOut);
  });

  it("says how long an idea may be when the text is far too long", async () => {
    // arrange
    respondWith(413, { status: 413 });

    // act
    const planted = await plantIdea("an-access-token", "p-1", "蛙".repeat(5000));

    // assert
    expect(planted).toEqual({ problem: "an idea has at most 500 characters" });
  });

  it("still gives a reason when frogly's refusal has none", async () => {
    // arrange
    vi.stubGlobal("fetch", vi.fn(async () => new Response("not JSON", { status: 400 })));

    // act
    const planted = await plantIdea("an-access-token", "p-1", "an idea");

    // assert
    expect(planted).toEqual({ problem: "frogly can't use that idea" });
  });

  it("fails when frogly can't answer", async () => {
    // arrange
    respondWith(500, { status: 500 });

    // act
    const planted = plantIdea("an-access-token", "p-1", "an idea");

    // assert
    await expect(planted).rejects.toThrow("500");
  });
});
