import { describe, expect, it, vi } from "vitest";

import { LoggedOut, type Project } from "./api";
import { addProject, loadProjects } from "./projects";

const garden: Project = { id: "p-1", name: "Garden" };
const token = async () => "an-access-token";

describe("addProject", () => {
  it("gives the new list once the project is saved", async () => {
    // arrange
    const api = {
      createProject: vi.fn(async () => ({ project: garden })),
      fetchProjects: vi.fn(async () => [garden]),
    };

    // act
    const outcome = await addProject(token, "Garden", api);

    // assert
    expect(outcome).toEqual({ kind: "saved", projects: [garden] });
  });

  it("gives frogly's reason when the name can't be used", async () => {
    // arrange
    const api = {
      createProject: vi.fn(async () => ({ problem: "a project needs a name" })),
      fetchProjects: vi.fn(async () => []),
    };

    // act
    const outcome = await addProject(token, "  ", api);

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "a project needs a name" });
    expect(api.fetchProjects).not.toHaveBeenCalled();
  });

  it("says the project is saved when only the list fails", async () => {
    // arrange
    const api = {
      createProject: vi.fn(async () => ({ project: garden })),
      fetchProjects: vi.fn(async () => {
        throw new Error("frogly answered 503");
      }),
    };

    // act
    const outcome = await addProject(token, "Garden", api);

    // assert
    expect(outcome).toEqual({ kind: "savedButNotListed" });
  });

  it("says the save failed when it did", async () => {
    // arrange
    const api = {
      createProject: vi.fn(async () => {
        throw new Error("frogly answered 500");
      }),
      fetchProjects: vi.fn(async () => []),
    };

    // act
    const outcome = await addProject(token, "Garden", api);

    // assert
    expect(outcome).toEqual({ kind: "notSaved" });
  });

  it("notices that the login ended", async () => {
    // arrange
    const api = {
      createProject: vi.fn(async () => {
        throw new LoggedOut();
      }),
      fetchProjects: vi.fn(async () => []),
    };

    // act
    const outcome = await addProject(token, "Garden", api);

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });
});

describe("loadProjects", () => {
  it("gives your projects", async () => {
    // act
    const outcome = await loadProjects(token, { fetchProjects: vi.fn(async () => [garden]) });

    // assert
    expect(outcome).toEqual({ kind: "listed", projects: [garden] });
  });

  it("notices that the login ended", async () => {
    // act
    const outcome = await loadProjects(token, {
      fetchProjects: vi.fn(async () => {
        throw new LoggedOut();
      }),
    });

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });

  it("says when the list can't be loaded", async () => {
    // act
    const outcome = await loadProjects(token, {
      fetchProjects: vi.fn(async () => {
        throw new Error("frogly answered 503");
      }),
    });

    // assert
    expect(outcome).toEqual({ kind: "failed" });
  });
});
