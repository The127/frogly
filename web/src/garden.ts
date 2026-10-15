import { type Idea, LoggedOut, NotFound, type Planted, type Project, fetchGarden, fetchProjects, plantIdea } from "./api";

type AccessToken = () => Promise<string>;

export type GardenLoaded =
  | { kind: "listed"; project: Project; ideas: Idea[] }
  | { kind: "notFound" }
  | { kind: "failed" }
  | { kind: "loggedOut" };

// The project's name comes from your list, since frogly has no call for a
// single project yet.
export async function loadGarden(accessToken: AccessToken, projectId: string): Promise<GardenLoaded> {
  let token: string;
  try {
    token = await accessToken();
  } catch (error) {
    return failure(error);
  }

  // both calls run at once. When both fail, the outcome mustn't depend on
  // which failed first, so an ended login wins, then a missing project.
  const [projects, ideas] = await Promise.allSettled([fetchProjects(token), fetchGarden(token, projectId)]);
  if (projects.status === "rejected" || ideas.status === "rejected") {
    const kinds = [projects, ideas]
      .filter((settled) => settled.status === "rejected")
      .map((settled) => failure(settled.reason).kind);

    return kinds.includes("loggedOut") ? { kind: "loggedOut" } : kinds.includes("notFound") ? { kind: "notFound" } : { kind: "failed" };
  }

  const project = projects.value.find((candidate) => candidate.id === projectId);

  return project ? { kind: "listed", project, ideas: ideas.value } : { kind: "notFound" };
}

// savedButNotListed is its own outcome, since saying the planting failed
// would make the person try again and plant the idea twice.
export type IdeaAdded =
  | { kind: "saved"; ideas: Idea[] }
  | { kind: "savedButNotListed" }
  | { kind: "refused"; problem: string }
  | { kind: "notSaved" }
  | { kind: "notFound" }
  | { kind: "loggedOut" };

export async function addIdea(accessToken: AccessToken, projectId: string, text: string): Promise<IdeaAdded> {
  let planted: Planted;

  try {
    planted = await plantIdea(await accessToken(), projectId, text);
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notSaved" } : failed;
  }

  if ("problem" in planted) {
    return { kind: "refused", problem: planted.problem };
  }

  try {
    return { kind: "saved", ideas: await fetchGarden(await accessToken(), projectId) };
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "savedButNotListed" } : failed;
  }
}

function failure(error: unknown): { kind: "notFound" } | { kind: "failed" } | { kind: "loggedOut" } {
  if (error instanceof LoggedOut) {
    return { kind: "loggedOut" };
  }

  if (error instanceof NotFound) {
    return { kind: "notFound" };
  }

  console.error(error);

  return { kind: "failed" };
}

// The day an idea was planted, as the person reads dates. The locale and
// time zone are the browser's unless a test sets them.
export function plantedOn(plantedAt: string, locale?: string, timeZone?: string): string {
  return new Date(plantedAt).toLocaleDateString(locale, { day: "numeric", month: "short", year: "numeric", timeZone });
}
