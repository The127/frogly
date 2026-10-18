import { afterEach, describe, expect, it, vi } from "vitest";

import * as api from "./api";
import type { Result } from "./client";
import { respondWith, token } from "./testing";


afterEach(() => {
  vi.unstubAllGlobals();
});

const fair = { id: "p-1", name: "Summer fair", visibility: "public", joining: "open" };
const ana = { id: "u-2", name: "Ana Costa", email: "ana@school.org" };
const idea = { id: "i-1", text: "more stalls", plantedBy: "Karo", plantedAt: "2026-10-18T09:00:00Z", people: 1 };

// Each endpoint is one call: what it sends, and what it gives back for
// frogly's answer.
const endpoints: {
  name: string;
  run: () => Promise<Result<unknown>>;
  method: string;
  path: string;
  body?: unknown;
  status?: number;
  answer?: unknown;
  value: unknown;
}[] = [
  { name: "config", run: () => api.config(), method: "GET", path: "/api/config", answer: { issuer: "i", clientId: "c" }, value: { issuer: "i", clientId: "c" } },
  { name: "me", run: () => api.me(token), method: "GET", path: "/api/me", answer: { name: "Karo" }, value: { name: "Karo" } },
  { name: "projects", run: () => api.projects(token), method: "GET", path: "/api/projects", answer: [fair], value: [fair] },
  { name: "publicProjects", run: () => api.publicProjects(token), method: "GET", path: "/api/projects/public", answer: [], value: [] },
  { name: "project", run: () => api.project(token, "p/1"), method: "GET", path: "/api/projects/p%2F1", answer: { ...fair, members: 2 }, value: { ...fair, members: 2 } },
  {
    name: "createProject",
    run: () => api.createProject(token, "Summer fair", "public", "open"),
    method: "POST",
    path: "/api/projects",
    body: { name: "Summer fair", visibility: "public", joining: "open" },
    status: 201,
    answer: fair,
    value: fair,
  },
  {
    name: "changeVisibility",
    run: () => api.changeVisibility(token, "p-1", "private"),
    method: "PUT",
    path: "/api/projects/p-1/visibility",
    body: { visibility: "private" },
    answer: fair,
    value: fair,
  },
  {
    name: "changeJoining",
    run: () => api.changeJoining(token, "p-1", "request"),
    method: "PUT",
    path: "/api/projects/p-1/joining",
    body: { joining: "request" },
    answer: fair,
    value: fair,
  },
  { name: "join", run: () => api.join(token, "p-1"), method: "POST", path: "/api/projects/p-1/join", answer: fair, value: fair },
  { name: "leave", run: () => api.leave(token, "p-1"), method: "POST", path: "/api/projects/p-1/leave", status: 204, value: undefined },
  { name: "askToJoin", run: () => api.askToJoin(token, "p-1"), method: "POST", path: "/api/projects/p-1/requests", answer: { asked: true }, value: true },
  { name: "requests", run: () => api.requests(token, "p-1"), method: "GET", path: "/api/projects/p-1/requests", answer: [], value: [] },
  { name: "accept", run: () => api.accept(token, "p-1", "u/2"), method: "POST", path: "/api/projects/p-1/requests/u%2F2/accept", answer: ana, value: undefined },
  { name: "decline", run: () => api.decline(token, "p-1", "u/2"), method: "DELETE", path: "/api/projects/p-1/requests/u%2F2", status: 204, value: undefined },
  {
    name: "searchPeople",
    run: () => api.searchPeople(token, "p-1", "ana & co"),
    method: "GET",
    path: "/api/projects/p-1/people?search=ana+%26+co",
    answer: [ana],
    value: [ana],
  },
  {
    name: "addMember",
    run: () => api.addMember(token, "p-1", "u-2"),
    method: "POST",
    path: "/api/projects/p-1/members",
    body: { userId: "u-2" },
    answer: ana,
    value: ana,
  },
  { name: "garden", run: () => api.garden(token, "p-1"), method: "GET", path: "/api/projects/p-1/ideas", answer: [idea], value: [idea] },
  {
    name: "plant",
    run: () => api.plant(token, "p-1", "more stalls"),
    method: "POST",
    path: "/api/projects/p-1/ideas",
    body: { text: "more stalls" },
    status: 201,
    answer: idea,
    value: idea,
  },
  { name: "meToo", run: () => api.meToo(token, "p-1", "i/1"), method: "POST", path: "/api/projects/p-1/ideas/i%2F1/metoo", answer: idea, value: idea },
];

describe.each(endpoints)("$name", ({ name, run, method, path, body, status, answer, value }) => {
  it("sends the call and gives frogly's answer", async () => {
    // arrange
    respondWith(status ?? 200, answer);

    // act
    const result = await run();

    // assert
    expect(result).toEqual({ ok: true, value });
    const [sentPath, init] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit];
    expect(sentPath).toBe(path);
    expect(init.method).toBe(method);
    expect(init.body).toBe(body === undefined ? undefined : JSON.stringify(body));
  });

  it("sends the login, and says when it sends JSON", async () => {
    // arrange
    respondWith(status ?? 200, answer);

    // act
    await run();

    // assert
    const [, init] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit];
    const headers: Record<string, string> = {};
    if (name !== "config") {
      headers.Authorization = "Bearer an-access-token";
    }

    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
    }

    expect(init.headers).toEqual(headers);
  });
});

describe("refusals", () => {
  it.each([
    { name: "createProject", run: () => api.createProject(token, "", undefined, undefined), problem: "frogly can't create that project" },
    { name: "searchPeople", run: () => api.searchPeople(token, "p-1", ""), problem: "frogly can't search for that" },
    { name: "plant", run: () => api.plant(token, "p-1", ""), problem: "frogly can't use that idea" },
  ])("$name gives a reason when frogly gives none", async ({ run, problem }) => {
    // arrange
    respondWith(400, { status: 400 });

    // act
    const result = await run();

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "refused", problem } });
  });

  it("plant says the limit when the idea is far too long to read", async () => {
    // arrange
    respondWith(413, { status: 413 });

    // act
    const result = await api.plant(token, "p-1", "x".repeat(10_000));

    // assert
    expect(result).toEqual({ ok: false, failure: { kind: "refused", problem: `an idea has at most ${api.maxIdeaLength} characters` } });
  });
});
