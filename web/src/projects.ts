import {
  type Created,
  type Project,
  type SessionEnded,
  type Visibility,
  createProject,
  fetchProjects,
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

// savedButNotListed is its own outcome, since saying the save failed would
// make the person try again and create the project twice.
export type Added =
  | { kind: "saved"; projects: Project[] }
  | { kind: "savedButNotListed" }
  | { kind: "refused"; problem: string }
  | { kind: "notSaved" }
  | SessionEnded;

export async function addProject(accessToken: AccessToken, name: string, visibility: Visibility | undefined): Promise<Added> {
  let created: Created;

  try {
    created = await createProject(await accessToken(), name, visibility);
  } catch (error) {
    const ended = sessionEnded(error);
    if (ended) {
      return ended;
    }

    console.error(error);

    return { kind: "notSaved" };
  }

  if ("problem" in created) {
    return { kind: "refused", problem: created.problem };
  }

  const loaded = await loadProjects(accessToken);
  switch (loaded.kind) {
    case "listed":
      return { kind: "saved", projects: loaded.projects };
    case "loggedOut":
    case "notAllowed":
      return { kind: loaded.kind };
    case "failed":
      return { kind: "savedButNotListed" };
  }
}
