import { beforeEach, describe, expect, it, vi } from "vitest";

import { type Idea, LoggedOut, NotFound, type Project, fetchGarden, fetchProjects, plantIdea } from "./api";
import { addIdea, loadGarden, plantedOn } from "./garden";

vi.mock("./api", async (actual) => ({
  ...(await actual<typeof import("./api")>()),
  fetchGarden: vi.fn(),
  fetchProjects: vi.fn(),
  plantIdea: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
});

const project: Project = { id: "p-1", name: "Garden" };
const idea: Idea = { id: "i-1", text: "search by any word", plantedBy: "Karo", plantedAt: "2026-10-15T21:00:00Z" };
const token = async () => "an-access-token";

describe("loadGarden", () => {
  it("gives the project and its ideas", async () => {
    // arrange
    vi.mocked(fetchProjects).mockResolvedValue([{ id: "p-2", name: "Kitchen" }, project]);
    vi.mocked(fetchGarden).mockResolvedValue([idea]);

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "listed", project, ideas: [idea] });
    expect(fetchGarden).toHaveBeenCalledWith("an-access-token", "p-1");
  });

  it("says the project isn't there when frogly says so", async () => {
    // arrange
    vi.mocked(fetchProjects).mockResolvedValue([project]);
    vi.mocked(fetchGarden).mockRejectedValue(new NotFound());

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "notFound" });
  });

  it("says the project isn't there when it's missing from your list", async () => {
    // arrange
    vi.mocked(fetchProjects).mockResolvedValue([]);
    vi.mocked(fetchGarden).mockResolvedValue([]);

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "notFound" });
  });

  it("says loading failed when frogly can't answer", async () => {
    // arrange
    vi.mocked(fetchProjects).mockResolvedValue([project]);
    vi.mocked(fetchGarden).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "failed" });
  });

  it("says loading failed when only the list fails", async () => {
    // arrange
    vi.mocked(fetchProjects).mockRejectedValue(new Error("frogly answered 503"));
    vi.mocked(fetchGarden).mockResolvedValue([idea]);

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "failed" });
  });

  it.each([
    { name: "the list", projects: new LoggedOut(), garden: new Error("frogly answered 503") },
    { name: "the garden", projects: new Error("frogly answered 503"), garden: new LoggedOut() },
    { name: "both", projects: new LoggedOut(), garden: new NotFound() },
  ])("says the login ended when $name says so, whatever else fails", async ({ projects, garden }) => {
    // arrange
    vi.mocked(fetchProjects).mockRejectedValue(projects);
    vi.mocked(fetchGarden).mockRejectedValue(garden);

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });

  it("says the project isn't there even when the list fails too", async () => {
    // arrange
    vi.mocked(fetchProjects).mockRejectedValue(new Error("frogly answered 503"));
    vi.mocked(fetchGarden).mockRejectedValue(new NotFound());

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "notFound" });
  });
});

describe("addIdea", () => {
  it("gives the new garden once the idea is planted", async () => {
    // arrange
    vi.mocked(plantIdea).mockResolvedValue({ idea });
    vi.mocked(fetchGarden).mockResolvedValue([idea]);

    // act
    const outcome = await addIdea(token, "p-1", "search by any word");

    // assert
    expect(outcome).toEqual({ kind: "saved", ideas: [idea] });
    expect(plantIdea).toHaveBeenCalledWith("an-access-token", "p-1", "search by any word");
  });

  it("gives frogly's reason when the text can't be used", async () => {
    // arrange
    vi.mocked(plantIdea).mockResolvedValue({ problem: "an idea needs a text" });

    // act
    const outcome = await addIdea(token, "p-1", "  ");

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "an idea needs a text" });
    expect(fetchGarden).not.toHaveBeenCalled();
  });

  it("says the idea is planted when only the garden fails", async () => {
    // arrange
    vi.mocked(plantIdea).mockResolvedValue({ idea });
    vi.mocked(fetchGarden).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await addIdea(token, "p-1", "an idea");

    // assert
    expect(outcome).toEqual({ kind: "savedButNotListed" });
  });

  it.each([
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
  ])("says so when $name before the garden could be listed again", async ({ error, kind }) => {
    // arrange
    vi.mocked(plantIdea).mockResolvedValue({ idea });
    vi.mocked(fetchGarden).mockRejectedValue(error);

    // act
    const outcome = await addIdea(token, "p-1", "an idea");

    // assert
    expect(outcome).toEqual({ kind });
  });

  it("says planting failed when it did", async () => {
    // arrange
    vi.mocked(plantIdea).mockRejectedValue(new Error("frogly answered 500"));

    // act
    const outcome = await addIdea(token, "p-1", "an idea");

    // assert
    expect(outcome).toEqual({ kind: "notSaved" });
  });

  it("says the project isn't there anymore", async () => {
    // arrange
    vi.mocked(plantIdea).mockRejectedValue(new NotFound());

    // act
    const outcome = await addIdea(token, "p-1", "an idea");

    // assert
    expect(outcome).toEqual({ kind: "notFound" });
  });

  it("says the login ended", async () => {
    // arrange
    vi.mocked(plantIdea).mockRejectedValue(new LoggedOut());

    // act
    const outcome = await addIdea(token, "p-1", "an idea");

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });
});

describe("plantedOn", () => {
  it("gives the day in the person's own time zone", async () => {
    // act
    const day = plantedOn("2026-10-15T23:30:00Z", "en-GB", "Europe/Berlin");

    // assert
    expect(day).toBe("16 Oct 2026");
  });
});
