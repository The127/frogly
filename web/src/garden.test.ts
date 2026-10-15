import { beforeEach, describe, expect, it, vi } from "vitest";

import { type Idea, LoggedOut, NotFound, type Project, fetchGarden, fetchProjects, plantIdea } from "./api";
import { addIdea, ideaLength, loadGarden, plantedOn } from "./garden";

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
    vi.mocked(fetchGarden).mockResolvedValue([idea]);
    vi.mocked(fetchProjects).mockResolvedValue([{ id: "p-2", name: "Kitchen" }, project]);

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "listed", project, ideas: [idea] });
    expect(fetchGarden).toHaveBeenCalledWith("an-access-token", "p-1");
  });

  it("says the project isn't there when frogly says so", async () => {
    // arrange
    vi.mocked(fetchGarden).mockRejectedValue(new NotFound());

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "notFound" });
    expect(fetchProjects).not.toHaveBeenCalled();
  });

  it("says the project isn't there when it's missing from your list", async () => {
    // arrange
    vi.mocked(fetchGarden).mockResolvedValue([]);
    vi.mocked(fetchProjects).mockResolvedValue([]);

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "notFound" });
  });

  it("says loading failed when the garden fails", async () => {
    // arrange
    vi.mocked(fetchGarden).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "failed" });
  });

  it("says loading failed when the list fails", async () => {
    // arrange
    vi.mocked(fetchGarden).mockResolvedValue([idea]);
    vi.mocked(fetchProjects).mockRejectedValue(new Error("frogly answered 503"));

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "failed" });
  });

  it("says the login ended", async () => {
    // arrange
    vi.mocked(fetchGarden).mockRejectedValue(new LoggedOut());

    // act
    const outcome = await loadGarden(token, "p-1");

    // assert
    expect(outcome).toEqual({ kind: "loggedOut" });
  });
});

describe("addIdea", () => {
  it("gives the planted idea", async () => {
    // arrange
    vi.mocked(plantIdea).mockResolvedValue({ idea });

    // act
    const outcome = await addIdea(token, "p-1", "search by any word");

    // assert
    expect(outcome).toEqual({ kind: "saved", idea });
    expect(plantIdea).toHaveBeenCalledWith("an-access-token", "p-1", "search by any word");
  });

  it("gives frogly's reason when the text can't be used", async () => {
    // arrange
    vi.mocked(plantIdea).mockResolvedValue({ problem: "an idea needs a text" });

    // act
    const outcome = await addIdea(token, "p-1", "  ");

    // assert
    expect(outcome).toEqual({ kind: "refused", problem: "an idea needs a text" });
  });

  it.each([
    { name: "planting failed", error: new Error("frogly answered 500"), kind: "notSaved" },
    { name: "the project is gone", error: new NotFound(), kind: "notFound" },
    { name: "the login ended", error: new LoggedOut(), kind: "loggedOut" },
  ])("says so when $name", async ({ error, kind }) => {
    // arrange
    vi.mocked(plantIdea).mockRejectedValue(error);

    // act
    const outcome = await addIdea(token, "p-1", "an idea");

    // assert
    expect(outcome).toEqual({ kind });
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

describe("ideaLength", () => {
  it("counts characters like frogly, after trimming", async () => {
    // act
    const length = ideaLength("  grüne Ideen 🌱  ");

    // assert
    expect(length).toBe(13);
  });
});
