export interface Config {
  issuer: string;
  clientId: string;
}

export interface Me {
  name: string;
}

export type Visibility = "public" | "private";

export type Joining = "open" | "request" | "closed";

export interface Project {
  id: string;
  name: string;
  visibility: Visibility;
  joining: Joining;
}

// PublicProject is what someone outside a public project sees of it.
export interface PublicProject {
  id: string;
  name: string;
  joining: Joining;
  // whether you asked to join and wait for an answer
  asked: boolean;
}

// Person is someone who has logged in to frogly.
export interface Person {
  id: string;
  name: string;
  email: string;
}

// WaitingRequest is someone who asked to join, as members see them.
export interface WaitingRequest {
  id: string;
  name: string;
  email: string;
  // an RFC 3339 time in UTC
  askedAt: string;
}

// frogly's limit for an idea, in characters
export const maxIdeaLength = 500;

export interface Idea {
  id: string;
  text: string;
  plantedBy: string;
  // an RFC 3339 time in UTC
  plantedAt: string;
  // the planter and everyone who said me too
  people: number;
}

// LoggedOut means the login ended, and trying again won't help until the
// person logs in again.
export class LoggedOut extends Error {
  constructor() {
    super("the login ended");
    this.name = "LoggedOut";
  }
}

// NotAllowed means the login works, but it doesn't give the person frogly's
// role, so only whoever runs frogly can help.
export class NotAllowed extends Error {
  constructor() {
    super("this login may not use frogly");
    this.name = "NotAllowed";
  }
}

export type SessionEnded = { kind: "loggedOut" } | { kind: "notAllowed" };

export function sessionEnded(error: unknown): SessionEnded | undefined {
  if (error instanceof LoggedOut) {
    return { kind: "loggedOut" };
  }

  if (error instanceof NotAllowed) {
    return { kind: "notAllowed" };
  }

  return undefined;
}

// failure says how a call about one project failed, so its page can say
// what to do about it.
export function failure(error: unknown): { kind: "notFound" } | { kind: "failed" } | SessionEnded {
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

// NotFound means the project doesn't exist or the person isn't a member of
// it. frogly doesn't say which.
export class NotFound extends Error {
  constructor() {
    super("the project isn't there");
    this.name = "NotFound";
  }
}

export async function fetchConfig(): Promise<Config> {
  const response = await fetch("/api/config");
  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked where to log in`);
  }

  return (await response.json()) as Config;
}

// Every call that needs the login goes through here, so a refused token
// always ends up as LoggedOut, and a login without frogly's role as
// NotAllowed.
async function authorized(path: string, accessToken: string, init: RequestInit = {}): Promise<Response> {
  const response = await fetch(path, {
    ...init,
    headers: { Authorization: `Bearer ${accessToken}`, ...init.headers },
  });
  if (response.status === 401) {
    throw new LoggedOut();
  }

  if (response.status === 403) {
    throw new NotAllowed();
  }

  return response;
}

// null means frogly refused the token, so the person has to log in again.
// NotAllowed means the login has no frogly role. Any other failure is
// frogly's problem, not the person's.
export async function fetchMe(accessToken: string): Promise<Me | null> {
  let response: Response;
  try {
    response = await authorized("/api/me", accessToken);
  } catch (error) {
    if (error instanceof LoggedOut) {
      return null;
    }

    throw error;
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked who is logged in`);
  }

  return (await response.json()) as Me;
}

export async function fetchProjects(accessToken: string): Promise<Project[]> {
  const response = await authorized("/api/projects", accessToken);
  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for your projects`);
  }

  return (await response.json()) as Project[];
}

export async function changeVisibility(accessToken: string, projectId: string, visibility: Visibility): Promise<Project> {
  return putSetting(accessToken, projectId, "visibility", { visibility }, "change a project's visibility");
}

// Either the people found, or frogly's reason for refusing the search,
// which the person can act on.
export async function searchPeople(
  accessToken: string,
  projectId: string,
  text: string,
): Promise<{ people: Person[] } | { problem: string }> {
  const search = new URLSearchParams({ search: text }).toString();
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}/people?${search}`, accessToken);
  if (response.status === 404) {
    throw new NotFound();
  }

  if (response.status === 400) {
    return { problem: await refusal(response, "frogly can't search for that") };
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to search for people`);
  }

  return { people: (await response.json()) as Person[] };
}

// the person comes from frogly's own search, so a refusal is frogly's
// problem, not something the person can fix
export async function addMember(accessToken: string, projectId: string, userId: string): Promise<Person> {
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}/members`, accessToken, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ userId }),
  });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to add someone to a project`);
  }

  return (await response.json()) as Person;
}

export async function sayMeToo(accessToken: string, projectId: string, ideaId: string): Promise<Idea> {
  const path = `${gardenPath(projectId)}/${encodeURIComponent(ideaId)}/metoo`;
  const response = await authorized(path, accessToken, { method: "POST" });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to say me too`);
  }

  return (await response.json()) as Idea;
}

export async function changeJoining(accessToken: string, projectId: string, joining: Joining): Promise<Project> {
  return putSetting(accessToken, projectId, "joining", { joining }, "change who can join a project");
}

// putSetting changes one setting of a project a member may change, and
// gives the project back as it is now
async function putSetting(accessToken: string, projectId: string, setting: string, body: object, what: string): Promise<Project> {
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}/${setting}`, accessToken, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to ${what}`);
  }

  return (await response.json()) as Project;
}

export async function fetchRequests(accessToken: string, projectId: string): Promise<WaitingRequest[]> {
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}/requests`, accessToken);
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for the requests to join a project`);
  }

  return (await response.json()) as WaitingRequest[];
}

export async function acceptRequest(accessToken: string, projectId: string, userId: string): Promise<Person> {
  const response = await authorized(requestPath(projectId, userId) + "/accept", accessToken, { method: "POST" });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to say yes to a request`);
  }

  return (await response.json()) as Person;
}

export async function declineRequest(accessToken: string, projectId: string, userId: string): Promise<void> {
  const response = await authorized(requestPath(projectId, userId), accessToken, { method: "DELETE" });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to say no to a request`);
  }
}

function requestPath(projectId: string, userId: string): string {
  return `/api/projects/${encodeURIComponent(projectId)}/requests/${encodeURIComponent(userId)}`;
}

// a member asking has nothing to wait for, so the answer can be false
export async function askToJoin(accessToken: string, projectId: string): Promise<boolean> {
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}/requests`, accessToken, { method: "POST" });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for a request to join a project`);
  }

  return ((await response.json()) as { asked: boolean }).asked;
}

export async function joinProject(accessToken: string, projectId: string): Promise<Project> {
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}/join`, accessToken, { method: "POST" });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to join a project`);
  }

  return (await response.json()) as Project;
}

export async function fetchPublicProjects(accessToken: string): Promise<PublicProject[]> {
  const response = await authorized("/api/projects/public", accessToken);
  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for the public projects`);
  }

  return (await response.json()) as PublicProject[];
}

export async function fetchProject(accessToken: string, projectId: string): Promise<Project> {
  const response = await authorized(`/api/projects/${encodeURIComponent(projectId)}`, accessToken);
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for a project`);
  }

  return (await response.json()) as Project;
}

// Either the new project, or frogly's reason for refusing the name, which
// the person can act on. Any other failure is frogly's problem.
export type Created = { project: Project } | { problem: string };

// visibility and joining are undefined until the person chooses, and
// frogly's answer then says that a choice is missing
export async function createProject(
  accessToken: string,
  name: string,
  visibility: Visibility | undefined,
  joining: Joining | undefined,
): Promise<Created> {
  const response = await authorized("/api/projects", accessToken, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, visibility, joining }),
  });
  if (response.status === 400) {
    return { problem: await refusal(response, "frogly can't create that project") };
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to create a project`);
  }

  return { project: (await response.json()) as Project };
}

// the reason in frogly's problem, or fallback when it has none
async function refusal(response: Response, fallback: string): Promise<string> {
  const problem = (await response.json().catch(() => ({}))) as { detail?: string };

  return problem.detail ?? fallback;
}

function gardenPath(projectId: string): string {
  return `/api/projects/${encodeURIComponent(projectId)}/ideas`;
}

export async function fetchGarden(accessToken: string, projectId: string): Promise<Idea[]> {
  const response = await authorized(gardenPath(projectId), accessToken);
  if (response.status === 404) {
    throw new NotFound();
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for a garden`);
  }

  return (await response.json()) as Idea[];
}

// Either the new idea, or frogly's reason for refusing the text, which the
// person can act on. Any other failure is frogly's problem.
export type Planted = { idea: Idea } | { problem: string };

export async function plantIdea(accessToken: string, projectId: string, text: string): Promise<Planted> {
  const response = await authorized(gardenPath(projectId), accessToken, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text }),
  });
  if (response.status === 404) {
    throw new NotFound();
  }

  if (response.status === 400) {
    return { problem: await refusal(response, "frogly can't use that idea") };
  }

  // frogly stops reading a body far longer than an idea may be
  if (response.status === 413) {
    return { problem: `an idea has at most ${maxIdeaLength} characters` };
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to plant an idea`);
  }

  return { idea: (await response.json()) as Idea };
}
