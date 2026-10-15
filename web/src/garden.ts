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
  try {
    const token = await accessToken();
    const [projects, ideas] = await Promise.all([fetchProjects(token), fetchGarden(token, projectId)]);
    const project = projects.find((candidate) => candidate.id === projectId);

    return project ? { kind: "listed", project, ideas } : { kind: "notFound" };
  } catch (error) {
    return failure(error);
  }
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
