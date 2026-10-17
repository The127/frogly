import { NotFound, type Project, type SessionEnded, type Visibility, changeVisibility, fetchProject, sessionEnded } from "./api";

type AccessToken = () => Promise<string>;

export type SettingsLoaded = { kind: "loaded"; project: Project } | { kind: "notFound" } | { kind: "failed" } | SessionEnded;

export async function loadSettings(accessToken: AccessToken, projectId: string): Promise<SettingsLoaded> {
  try {
    return { kind: "loaded", project: await fetchProject(await accessToken(), projectId) };
  } catch (error) {
    return failure(error);
  }
}

export type VisibilitySwitched = { kind: "changed"; project: Project } | { kind: "notChanged" } | { kind: "notFound" } | SessionEnded;

export async function switchVisibility(accessToken: AccessToken, projectId: string, visibility: Visibility): Promise<VisibilitySwitched> {
  try {
    return { kind: "changed", project: await changeVisibility(await accessToken(), projectId, visibility) };
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notChanged" } : failed;
  }
}

function failure(error: unknown): { kind: "notFound" } | { kind: "failed" } | SessionEnded {
  const ended = sessionEnded(error);
  if (ended) {
    return ended;
  }

  if (error instanceof NotFound) {
    return { kind: "notFound" };
  }

  console.error(error);

  return { kind: "failed" };
}
