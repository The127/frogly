import {
  type Created,
  type Project,
  type PublicProject,
  type Joining,
  type SessionEnded,
  type Visibility,
  createProject,
  fetchProjects,
  fetchPublicProjects,
  sessionEnded,
} from "./api";

type AccessToken = () => Promise<string>;

export type Loaded =
  | { kind: "listed"; projects: Project[] }
  | { kind: "failed" }
  | SessionEnded;

export async function loadProjects(accessToken: AccessToken): Promise<Loaded> {
  try {
    return { kind: "listed", projects: await fetchProjects(await accessToken()) };
  } catch (error) {
    const ended = sessionEnded(error);
    if (ended) {
      return ended;
    }

    console.error(error);

    return { kind: "failed" };
  }
}

export type Added =
  | { kind: "saved"; project: Project }
  | { kind: "refused"; problem: string }
  | { kind: "notSaved" }
  | SessionEnded;

export async function addProject(
  accessToken: AccessToken,
  name: string,
  visibility: Visibility | undefined,
  joining: Joining | undefined,
): Promise<Added> {
  let created: Created;

  try {
    created = await createProject(await accessToken(), name, visibility, joining);
  } catch (error) {
    const ended = sessionEnded(error);
    if (ended) {
      return ended;
    }

    console.error(error);

    return { kind: "notSaved" };
  }

  return "problem" in created ? { kind: "refused", problem: created.problem } : { kind: "saved", project: created.project };
}

export type PublicLoaded = { kind: "listed"; projects: PublicProject[] } | { kind: "failed" } | SessionEnded;

export async function loadPublicProjects(accessToken: AccessToken): Promise<PublicLoaded> {
  try {
    return { kind: "listed", projects: await fetchPublicProjects(await accessToken()) };
  } catch (error) {
    const ended = sessionEnded(error);
    if (ended) {
      return ended;
    }

    console.error(error);

    return { kind: "failed" };
  }
}

// nameFrom reads the name the home page passed on. sv-router's route.search
// would turn a name like "2024" or "true" into a number or a boolean.
export function nameFrom(search: string): string {
  return new URLSearchParams(search).get("name") ?? "";
}
