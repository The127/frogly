import {
  type Created,
  type Joining,
  type Project,
  type PublicProject,
  type SessionEnded,
  type Visibility,
  createProject,
  fetchProjects,
  failure,
  fetchPublicProjects,
  joinProject,
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
    // the form keeps a join choice after a switch to private, which frogly
    // would refuse
    created = await createProject(await accessToken(), name, visibility, visibility === "public" ? joining : undefined);
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

export type Joined = { kind: "joined"; project: Project } | { kind: "notJoined" } | { kind: "notFound" } | SessionEnded;

// notFound means the project is closed, private or gone by now
export async function joinPublicProject(accessToken: AccessToken, projectId: string): Promise<Joined> {
  try {
    return { kind: "joined", project: await joinProject(await accessToken(), projectId) };
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notJoined" } : failed;
  }
}

// nameFrom reads the name the home page passed on. sv-router's route.search
// would turn a name like "2024" or "true" into a number or a boolean.
export function nameFrom(search: string): string {
  return new URLSearchParams(search).get("name") ?? "";
}
