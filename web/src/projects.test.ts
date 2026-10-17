import { beforeEach, describe, expect, it, vi } from "vitest";

import { LoggedOut, NotAllowed, type Project, createProject, fetchProjects } from "./api";
import { addProject, loadProjects } from "./projects";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  createProject: vi.fn(),
  fetchProjects: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
});

const garden: Project = { id: "p-1", name: "Garden" };
const token = async () => "an-access-token";

describe("addProject", () => {
  it("gives the new list once the project is saved", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });
    vi.mocked(fetchProjects).mockResolvedValue([garden]);

    // act
    const outcome = await addProject(token, "Garden");

    // assert
    expect(outcome).toEqual({ kind: "saved", projects: [garden] });
  });

  it("gives frogly's reason when the name can't be used", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ problem: "a project needs a name" });

    // act
    const outcome = await addProject(token, "  ");

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "a project needs a name" });
    expect(fetchProjects).not.toHaveBeenCalled();
  });

  it("says the project is saved when only the list fails", async () => {
    // arrange
    vi.mocked(createProject).mockResolvedValue({ project: garden });
    vi.mocked(fetchProjects).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await addProject(token, "Garden");

    // assert
    expect(outcome).toEqual({ kind: "savedButNotListed" });
  });

  it("says the save failed when it did", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new Error("frogly answered 500"));

    // act
    const outcome = await addProject(token, "Garden");

    // assert
    expect(outcome).toEqual({ kind: "notSaved" });
  });

  it("notices that the login ended", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new LoggedOut());

    // act
    const outcome = await addProject(token, "Garden");

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });

  it("notices that the login lost frogly's role", async () => {
    // arrange
    vi.mocked(createProject).mockRejectedValue(new NotAllowed());

    // act
    const outcome = await addProject(token, "Garden");

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
