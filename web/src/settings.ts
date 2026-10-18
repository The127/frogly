import {
  type Joining,
  type Person,
  type Project,
  type SessionEnded,
  type Visibility,
  type WaitingRequest,
  acceptRequest,
  addMember,
  changeJoining,
  changeVisibility,
  declineRequest,
  failure,
  fetchProject,
  fetchRequests,
  leaveProject,
  searchPeople,
} from "./api";

type AccessToken = () => Promise<string>;

export type SettingsLoaded =
  | { kind: "loaded"; project: Project; members: number }
  | { kind: "notFound" }
  | { kind: "failed" }
  | SessionEnded;

export async function loadSettings(accessToken: AccessToken, projectId: string): Promise<SettingsLoaded> {
  try {
    const { members, ...project } = await fetchProject(await accessToken(), projectId);

    return { kind: "loaded", project, members };
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

export type RequestsLoaded = { kind: "listed"; requests: WaitingRequest[] } | { kind: "failed" } | { kind: "notFound" } | SessionEnded;

export async function loadRequests(accessToken: AccessToken, projectId: string): Promise<RequestsLoaded> {
  try {
    return { kind: "listed", requests: await fetchRequests(await accessToken(), projectId) };
  } catch (error) {
    return failure(error);
  }
}

export type RequestAnswered = { kind: "answered" } | { kind: "notAnswered" } | { kind: "notFound" } | SessionEnded;

// notFound means the request is gone, answered by another member or since
export async function answerRequest(
  accessToken: AccessToken,
  projectId: string,
  userId: string,
  answer: "yes" | "no",
): Promise<RequestAnswered> {
  try {
    const token = await accessToken();
    if (answer === "yes") {
      await acceptRequest(token, projectId, userId);
    } else {
      await declineRequest(token, projectId, userId);
    }

    return { kind: "answered" };
  } catch (error) {
    const failed = failure(error);

    return failed.kind === "failed" ? { kind: "notAnswered" } : failed;
  }
}

export type Left = { kind: "left" } | { kind: "notLeft" } | SessionEnded;

export async function leave(accessToken: AccessToken, projectId: string): Promise<Left> {
  try {
    await leaveProject(await accessToken(), projectId);

    return { kind: "left" };
  } catch (error) {
    const failed = failure(error);
    switch (failed.kind) {
      case "failed":
        return { kind: "notLeft" };
      // not a member any more, like after leaving in another tab, is what
      // leaving was for
      case "notFound":
        return { kind: "left" };
      default:
        return failed;
    }
  }
}
