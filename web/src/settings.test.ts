import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  LoggedOut,
  NotAllowed,
  NotFound,
  type Person,
  type Project,
  addMember,
  changeVisibility,
  fetchProject,
  searchPeople,
} from "./api";
import { addPerson, findPeople, loadSettings, switchVisibility } from "./settings";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  changeVisibility: vi.fn(),
  fetchProject: vi.fn(),
  searchPeople: vi.fn(),
  addMember: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
});

const project: Project = { id: "p-1", name: "Garden", visibility: "private", joining: "closed" };
const token = async () => "an-access-token";

describe("loadSettings", () => {
  it("gives the project", async () => {
    // arrange
    vi.mocked(fetchProject).mockResolvedValue(project);

    // act
    const outcome = await loadSettings(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "loaded", project });
    expect(fetchProject).toHaveBeenCalledWith("an-access-token", "p-1");
  });

  it.each([
    { name: "it can't be loaded", error: new Error("frogly answered 503"), kind: "failed" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(fetchProject).mockRejectedValue(error);

    // act
    const outcome = await loadSettings(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind });
  });
});

describe("switchVisibility", () => {
  it("gives the changed project", async () => {
    // arrange
    vi.mocked(changeVisibility).mockResolvedValue({ ...project, visibility: "public" });

    // act
    const outcome = await switchVisibility(token, "p-1", "public");

    // assert
    expect(outcome).toEqual({ kind: "changed", project: { ...project, visibility: "public" } });
    expect(changeVisibility).toHaveBeenCalledWith("an-access-token", "p-1", "public");
  });

  it.each([
    { name: "the change failed", error: new Error("frogly answered 500"), kind: "notChanged" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(changeVisibility).mockRejectedValue(error);

    // act
    const outcome = await switchVisibility(token, "p-1", "public");

    // assert
    expect(outcome).toEqual({ kind });
  });
});

const ana: Person = { id: "u-2", name: "Ana Costa", email: "ana@school.org" };

describe("findPeople", () => {
  it("gives the people found", async () => {
    // arrange
    vi.mocked(searchPeople).mockResolvedValue({ people: [ana] });

    // act
    const outcome = await findPeople(token, "p-1", "ana");

    // assert
    expect(outcome).toEqual({ kind: "found", people: [ana] });
    expect(searchPeople).toHaveBeenCalledWith("an-access-token", "p-1", "ana");
  });

  it("gives frogly's reason when the search can't be used", async () => {
    // arrange
    vi.mocked(searchPeople).mockResolvedValue({ problem: "type part of a name or an email" });

    // act
    const outcome = await findPeople(token, "p-1", " ");

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "type part of a name or an email" });
  });

  it.each([
    { name: "the search failed", error: new Error("frogly answered 500"), kind: "failed" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(searchPeople).mockRejectedValue(error);

    // act
    const outcome = await findPeople(token, "p-1", "ana");

    // assert
    expect(outcome).toEqual({ kind });
  });
});

describe("addPerson", () => {
  it("gives the person added", async () => {
    // arrange
    vi.mocked(addMember).mockResolvedValue(ana);

    // act
    const outcome = await addPerson(token, "p-1", "u-2");

    // assert
    expect(outcome).toEqual({ kind: "added", person: ana });
    expect(addMember).toHaveBeenCalledWith("an-access-token", "p-1", "u-2");
  });

  it.each([
    { name: "adding failed", error: new Error("frogly answered 500"), kind: "notAdded" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(addMember).mockRejectedValue(error);

    // act
    const outcome = await addPerson(token, "p-1", "u-2");

    // assert
    expect(outcome).toEqual({ kind });
  });
});
