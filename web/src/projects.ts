import { type Created, LoggedOut, type Project, createProject, fetchProjects } from "./api";

type AccessToken = () => Promise<string>;

export type Loaded = { kind: "listed"; projects: Project[] } | { kind: "failed" } | { kind: "loggedOut" };

export async function loadProjects(accessToken: AccessToken): Promise<Loaded> {
  try {
    return { kind: "listed", projects: await fetchProjects(await accessToken()) };
  } catch (error) {
    if (error instanceof LoggedOut) {
      return { kind: "loggedOut" };
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
  | { kind: "loggedOut" };

export async function addProject(accessToken: AccessToken, name: string): Promise<Added> {
  let created: Created;

  try {
    created = await createProject(await accessToken(), name);
  } catch (error) {
    if (error instanceof LoggedOut) {
      return { kind: "loggedOut" };
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
      return { kind: "loggedOut" };
    case "failed":
      return { kind: "savedButNotListed" };
  }
}
