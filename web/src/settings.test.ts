import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  LoggedOut,
  NotAllowed,
  NotFound,
  type Person,
  type Project,
  acceptRequest,
  addMember,
  changeJoining,
  changeVisibility,
  declineRequest,
  fetchProject,
  fetchRequests,
  leaveProject,
  searchPeople,
} from "./api";
import { addPerson, answerRequest, findPeople, leave, loadRequests, loadSettings, switchJoining, switchVisibility } from "./settings";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  changeVisibility: vi.fn(),
  fetchProject: vi.fn(),
  searchPeople: vi.fn(),
  addMember: vi.fn(),
  changeJoining: vi.fn(),
  fetchRequests: vi.fn(),
  acceptRequest: vi.fn(),
  declineRequest: vi.fn(),
  leaveProject: vi.fn(),
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

describe("switchJoining", () => {
  it("gives the changed project", async () => {
    // arrange
    const open: Project = { ...project, visibility: "public", joining: "open" };
    vi.mocked(changeJoining).mockResolvedValue(open);

    // act
    const outcome = await switchJoining(token, "p-1", "open");

    // assert
    expect(outcome).toEqual({ kind: "changed", project: open });
    expect(changeJoining).toHaveBeenCalledWith("an-access-token", "p-1", "open");
  });

  it.each([
    { name: "the change failed", error: new Error("frogly answered 500"), kind: "notChanged" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(changeJoining).mockRejectedValue(error);

    // act
    const outcome = await switchJoining(token, "p-1", "open");

    // assert
    expect(outcome).toEqual({ kind });
  });
});

describe("loadRequests", () => {
  it("gives the waiting requests", async () => {
    // arrange
    const waiting = [{ id: "u-2", name: "Ana Costa", email: "ana@school.org", askedAt: "2026-10-18T09:00:00Z" }];
    vi.mocked(fetchRequests).mockResolvedValue(waiting);

    // act
    const outcome = await loadRequests(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "listed", requests: waiting });
  });

  it.each([
    { name: "they can't be loaded", error: new Error("frogly answered 503"), kind: "failed" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(fetchRequests).mockRejectedValue(error);

    // act
    const outcome = await loadRequests(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind });
  });
});

describe("answerRequest", () => {
  it("says yes", async () => {
    // arrange
    vi.mocked(acceptRequest).mockResolvedValue();

    // act
    const outcome = await answerRequest(token, "p-1", "u-2", "yes");

    // assert
    expect(outcome).toEqual({ kind: "answered" });
    expect(acceptRequest).toHaveBeenCalledWith("an-access-token", "p-1", "u-2");
    expect(declineRequest).not.toHaveBeenCalled();
  });

  it("says when the no failed", async () => {
    // arrange
    vi.mocked(declineRequest).mockRejectedValue(new Error("frogly answered 500"));

    // act
    const outcome = await answerRequest(token, "p-1", "u-2", "no");

    // assert
    expect(outcome).toEqual({ kind: "notAnswered" });
  });

  it("says no", async () => {
    // arrange
    vi.mocked(declineRequest).mockResolvedValue();

    // act
    const outcome = await answerRequest(token, "p-1", "u-2", "no");

    // assert
    expect(outcome).toEqual({ kind: "answered" });
    expect(declineRequest).toHaveBeenCalledWith("an-access-token", "p-1", "u-2");
    expect(acceptRequest).not.toHaveBeenCalled();
  });

  it.each([
    { name: "answering failed", error: new Error("frogly answered 500"), kind: "notAnswered" },
    { name: "the request is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(acceptRequest).mockRejectedValue(error);

    // act
    const outcome = await answerRequest(token, "p-1", "u-2", "yes");

    // assert
    expect(outcome).toEqual({ kind });
  });
});

describe("leave", () => {
  it("leaves the project", async () => {
    // arrange
    vi.mocked(leaveProject).mockResolvedValue();

    // act
    const outcome = await leave(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "left" });
    expect(leaveProject).toHaveBeenCalledWith("an-access-token", "p-1");
  });

  it.each([
    { name: "leaving failed", error: new Error("frogly answered 500"), kind: "notLeft" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the login lost frogly's role", error: new NotAllowed(), kind: "notAllowed" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(leaveProject).mockRejectedValue(error);

    // act
    const outcome = await leave(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind });
  });
});
