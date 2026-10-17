import {
  type Joining,
  type Person,
  type Project,
  type SessionEnded,
  type Visibility,
  addMember,
  changeJoining,
  changeVisibility,
  failure,
  fetchProject,
  searchPeople,
} from "./api";

type AccessToken = () => Promise<string>;

export type SettingsLoaded = { kind: "loaded"; project: Project } | { kind: "notFound" } | { kind: "failed" } | SessionEnded;

export async function loadSettings(accessToken: AccessToken, projectId: string): Promise<SettingsLoaded> {
  try {
    return { kind: "loaded", project: await fetchProject(await accessToken(), projectId) };
  } catch (error) {
    return failure(error);
  }
}

export type SettingChanged = { kind: "changed"; project: Project } | { kind: "notChanged" } | { kind: "notFound" } | SessionEnded;

export async function switchVisibility(accessToken: AccessToken, projectId: string, visibility: Visibility): Promise<SettingChanged> {
  return changeSetting(async () => changeVisibility(await accessToken(), projectId, visibility));
}

export async function switchJoining(accessToken: AccessToken, projectId: string, joining: Joining): Promise<SettingChanged> {
  return changeSetting(async () => changeJoining(await accessToken(), projectId, joining));
}

async function changeSetting(change: () => Promise<Project>): Promise<SettingChanged> {
  try {
    return { kind: "changed", project: await change() };
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notChanged" } : failed;
  }
}

export type PeopleFound =
  | { kind: "found"; people: Person[] }
  | { kind: "refused"; problem: string }
  | { kind: "failed" }
  | { kind: "notFound" }
  | SessionEnded;

export async function findPeople(accessToken: AccessToken, projectId: string, text: string): Promise<PeopleFound> {
  try {
    const found = await searchPeople(await accessToken(), projectId, text);

    return "problem" in found ? { kind: "refused", problem: found.problem } : { kind: "found", people: found.people };
  } catch (error) {
    return failure(error);
  }
}

export type PersonAdded = { kind: "added"; person: Person } | { kind: "notAdded" } | { kind: "notFound" } | SessionEnded;

export async function addPerson(accessToken: AccessToken, projectId: string, userId: string): Promise<PersonAdded> {
  try {
    return { kind: "added", person: await addMember(await accessToken(), projectId, userId) };
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notAdded" } : failed;
  }
}
