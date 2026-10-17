import { beforeEach, describe, expect, it, vi } from "vitest";

import { LoggedOut, NotAllowed, NotFound, type Project, changeVisibility, fetchProject } from "./api";
import { loadSettings, switchVisibility } from "./settings";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  changeVisibility: vi.fn(),
  fetchProject: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
});

const project: Project = { id: "p-1", name: "Garden", visibility: "private" };
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
