import {
  type Idea,
  type Planted,
  type Project,
  type SessionEnded,
  failure,
  fetchGarden,
  fetchProject,
  plantIdea,
} from "./api";

type AccessToken = () => Promise<string>;

export type GardenLoaded =
  | { kind: "listed"; project: Project; ideas: Idea[] }
  | { kind: "notFound" }
  | { kind: "failed" }
  | SessionEnded;

export async function loadGarden(accessToken: AccessToken, projectId: string): Promise<GardenLoaded> {
  try {
    const token = await accessToken();
    const project = await fetchProject(token, projectId);
    const ideas = await fetchGarden(token, projectId);

    return { kind: "listed", project, ideas };
  } catch (error) {
    return failure(error);
  }
}

export type IdeaAdded =
  | { kind: "saved"; idea: Idea }
  | { kind: "refused"; problem: string }
  | { kind: "notSaved" }
  | { kind: "notFound" }
  | SessionEnded;

export async function addIdea(accessToken: AccessToken, projectId: string, text: string): Promise<IdeaAdded> {
  let planted: Planted;

  try {
    planted = await plantIdea(await accessToken(), projectId, text);
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notSaved" } : failed;
  }

  return "problem" in planted ? { kind: "refused", problem: planted.problem } : { kind: "saved", idea: planted.idea };
}

// The day an idea was planted, as the person reads dates. The locale and
// time zone are the browser's unless a test sets them.
export function plantedOn(plantedAt: string, locale?: string, timeZone?: string): string {
  return new Date(plantedAt).toLocaleDateString(locale, { day: "numeric", month: "short", year: "numeric", timeZone });
}
