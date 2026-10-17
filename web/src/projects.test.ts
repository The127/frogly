import { beforeEach, describe, expect, it, vi } from "vitest";

import { LoggedOut, NotAllowed, type Project, createProject, fetchProjects, fetchPublicProjects } from "./api";
import { addProject, loadProjects, loadPublicProjects } from "./projects";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  createProject: vi.fn(),
  fetchProjects: vi.fn(),
  fetchPublicProjects: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
});

const garden: Project = { id: "p-1", name: "Garden", visibility: "private" };
const token = async () => "an-access-token";

describe("addProject", () => {
  it("gives the new list once the project is saved", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });
    vi.mocked(fetchProjects).mockResolvedValue([garden]);

    // act
    const outcome = await addProject(token, "Garden", "private");

    // assert
    expect(outcome).toEqual({ kind: "saved", projects: [garden] });
    expect(createProject).toHaveBeenCalledWith("an-access-token", "Garden", "private");
  });

  it("gives frogly's reason when the name can't be used", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ problem: "a project needs a name" });

    // act
    const outcome = await addProject(token, "  ", "private");

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "a project needs a name" });
    expect(fetchProjects).not.toHaveBeenCalled();
  });

  it("says the project is saved when only the list fails", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });
    vi.mocked(fetchProjects).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await addProject(token, "Garden", "private");

    // assert
    expect(outcome).toEqual({ kind: "savedButNotListed" });
  });

  it("notices a lost role when listing after the save", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });
    vi.mocked(fetchProjects).mockRejectedValue(new NotAllowed());

    // act
    const outcome = await addProject(token, "Garden", "private");

    // assert
    expect(outcome).toEqual({ kind: "notAllowed" });
  });

  it("says the save failed when it did", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new Error("frogly answered 500"));

    // act
    const outcome = await addProject(token, "Garden", "private");

    // assert
    expect(outcome).toEqual({ kind: "notSaved" });
  });

  it("notices that the login ended", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new LoggedOut());

    // act
    const outcome = await addProject(token, "Garden", "private");

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });

  it("notices that the login lost frogly's role", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new NotAllowed());

    // act
    const outcome = await addProject(token, "Garden", "private");

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
    vi.mocked(fetchPublicProjects).mockResolvedValue([{ id: "p-2", name: "Summer fair" }]);

    // act
    const outcome = await loadPublicProjects(token);

    // assert
    expect(outcome).toEqual({ kind: "listed", projects: [{ id: "p-2", name: "Summer fair" }] });
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
