import { beforeEach, describe, expect, it, vi } from "vitest";

import { LoggedOut, NotAllowed, NotFound, type Project, createProject, fetchProjects, fetchPublicProjects, joinProject } from "./api";
import { addProject, joinPublicProject, loadProjects, loadPublicProjects, nameFrom } from "./projects";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  createProject: vi.fn(),
  fetchProjects: vi.fn(),
  fetchPublicProjects: vi.fn(),
  joinProject: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
});

const garden: Project = { id: "p-1", name: "Garden", visibility: "private", joining: "closed" };
const token = async () => "an-access-token";

describe("addProject", () => {
  it("gives the new project once it's saved", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });

    // act
    const outcome = await addProject(token, "Garden", "private", undefined);

    // assert
    expect(outcome).toEqual({ kind: "saved", project: garden });
    expect(createProject).toHaveBeenCalledWith("an-access-token", "Garden", "private", undefined);
  });

  it("sends no choice of who can join for a private project", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });

    // act
    await addProject(token, "Garden", "private", "open");

    // assert
    expect(createProject).toHaveBeenCalledWith("an-access-token", "Garden", "private", undefined);
  });

  it("gives frogly's reason when the name can't be used", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ problem: "a project needs a name" });

    // act
    const outcome = await addProject(token, "  ", "private", undefined);

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "a project needs a name" });
  });

  it("says the save failed when it did", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new Error("frogly answered 500"));

    // act
    const outcome = await addProject(token, "Garden", "private", undefined);

    // assert
    expect(outcome).toEqual({ kind: "notSaved" });
  });

  it("notices that the login ended", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new LoggedOut());

    // act
    const outcome = await addProject(token, "Garden", "private", undefined);

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });

  it("notices that the login lost frogly's role", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new NotAllowed());

    // act
    const outcome = await addProject(token, "Garden", "private", undefined);

    // assert
    expect(outcome).toEqual({ kind: "notAllowed" });
  });
});

describe("loadProjects", () => {
  it("gives your projects", async () => {
    // arrange
    vi.mocked(fetchProjects).mockResolvedValue([garden]);

    // act
    const outcome = await loadProjects(token);

    // assert
    expect(outcome).toEqual({ kind: "listed", projects: [garden] });
  });

  it("notices that the login ended", async () => {
    // arrange
    vi.mocked(fetchProjects).mockRejectedValue(new LoggedOut());

    // act
    const outcome = await loadProjects(token);

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });

  it("notices that the login lost frogly's role", async () => {
    // arrange
    vi.mocked(fetchProjects).mockRejectedValue(new NotAllowed());

    // act
    const outcome = await loadProjects(token);

    // assert
    expect(outcome).toEqual({ kind: "notAllowed" });
  });

  it("says when the list can't be loaded", async () => {
    // arrange
    vi.mocked(fetchProjects).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await loadProjects(token);

    // assert
    expect(outcome).toEqual({ kind: "failed" });
  });
});

describe("loadPublicProjects", () => {
  it("gives the public projects of others", async () => {
    // arrange
    vi.mocked(fetchPublicProjects).mockResolvedValue([{ id: "p-2", name: "Summer fair", joining: "open" }]);

    // act
    const outcome = await loadPublicProjects(token);

    // assert
    expect(outcome).toEqual({ kind: "listed", projects: [{ id: "p-2", name: "Summer fair", joining: "open" }] });
  });

  it.each([
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
    { name: "the list can't be loaded", error: new Error("frogly answered 503"), kind: "failed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(fetchPublicProjects).mockRejectedValue(error);

    // act
    const outcome = await loadPublicProjects(token);

    // assert
    expect(outcome).toEqual({ kind });
  });
});

describe("nameFrom", () => {
  it.each(["Summer fair", "2024", "007", "true", " ", "a&b=c"])("keeps %j as the text it is", (name) => {
    // act
    const read = nameFrom("?" + new URLSearchParams({ name }).toString());

    // assert
    expect(read).toBe(name);
  });

  it("is empty without a name", () => {
    // act
    const read = nameFrom("");

    // assert
    expect(read).toBe("");
  });
});

describe("joinPublicProject", () => {
  it("gives the joined project", async () => {
    // arrange
    const fair: Project = { id: "p-2", name: "Summer fair", visibility: "public", joining: "open" };
    vi.mocked(joinProject).mockResolvedValue(fair);

    // act
    const outcome = await joinPublicProject(token, "p-2");

    // assert
    expect(outcome).toEqual({ kind: "joined", project: fair });
    expect(joinProject).toHaveBeenCalledWith("an-access-token", "p-2");
  });

  it.each([
    { name: "joining failed", error: new Error("frogly answered 500"), kind: "notJoined" },
    { name: "it can't be joined any more", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(joinProject).mockRejectedValue(error);

    // act
    const outcome = await joinPublicProject(token, "p-2");

    // assert
    expect(outcome).toEqual({ kind });
  });
});
