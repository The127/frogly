import { type Result, type Token, call, map } from "./client";

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

// ProjectWithMembers is one project as its members see it, with how many
// they are.
export interface ProjectWithMembers extends Project {
  members: number;
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

// WaitingRequest is someone who asked to join, as members see them. id is
// the asker's.
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

export function config(): Promise<Result<Config>> {
  return call("/api/config");
}

export function me(token: Token): Promise<Result<Me>> {
  return call("/api/me", { token });
}

export function projects(token: Token): Promise<Result<Project[]>> {
  return call("/api/projects", { token });
}

export function publicProjects(token: Token): Promise<Result<PublicProject[]>> {
  return call("/api/projects/public", { token });
}

export function project(token: Token, projectId: string): Promise<Result<ProjectWithMembers>> {
  return call(projectPath(projectId), { token });
}

// visibility and joining are undefined until the person chooses, and
// frogly's refusal then says that a choice is missing
export function createProject(
  token: Token,
  name: string,
  visibility: Visibility | undefined,
  joining: Joining | undefined,
): Promise<Result<Project>> {
  return call("/api/projects", {
    token,
    method: "POST",
    body: { name, visibility, joining },
    refused: "frogly can't create that project",
  });
}

export function changeVisibility(token: Token, projectId: string, visibility: Visibility): Promise<Result<Project>> {
  return call(`${projectPath(projectId)}/visibility`, { token, method: "PUT", body: { visibility } });
}

export function changeJoining(token: Token, projectId: string, joining: Joining): Promise<Result<Project>> {
  return call(`${projectPath(projectId)}/joining`, { token, method: "PUT", body: { joining } });
}

export function join(token: Token, projectId: string): Promise<Result<Project>> {
  return call(`${projectPath(projectId)}/join`, { token, method: "POST" });
}

export function leave(token: Token, projectId: string): Promise<Result<void>> {
  return call(`${projectPath(projectId)}/leave`, { token, method: "POST" });
}

// whether a request of yours waits now. A member has nothing to wait for.
export async function askToJoin(token: Token, projectId: string): Promise<Result<boolean>> {
  const asked = await call<{ asked: boolean }>(requestsPath(projectId), { token, method: "POST" });

  return map(asked, (answer) => answer.asked);
}

export function requests(token: Token, projectId: string): Promise<Result<WaitingRequest[]>> {
  return call(requestsPath(projectId), { token });
}

// frogly answers with the new member, which the frontend doesn't need
export async function accept(token: Token, projectId: string, askerId: string): Promise<Result<void>> {
  const accepted = await call(`${requestsPath(projectId)}/${encodeURIComponent(askerId)}/accept`, { token, method: "POST" });

  return map(accepted, () => undefined);
}

export function decline(token: Token, projectId: string, askerId: string): Promise<Result<void>> {
  return call(`${requestsPath(projectId)}/${encodeURIComponent(askerId)}`, { token, method: "DELETE" });
}

export function searchPeople(token: Token, projectId: string, text: string): Promise<Result<Person[]>> {
  const search = new URLSearchParams({ search: text }).toString();

  return call(`${projectPath(projectId)}/people?${search}`, { token, refused: "frogly can't search for that" });
}

// the person comes from frogly's own search, so a refusal is frogly's
// problem, not something the person can fix
export function addMember(token: Token, projectId: string, userId: string): Promise<Result<Person>> {
  return call(`${projectPath(projectId)}/members`, { token, method: "POST", body: { userId } });
}

export function garden(token: Token, projectId: string): Promise<Result<Idea[]>> {
  return call(gardenPath(projectId), { token });
}

export function plant(token: Token, projectId: string, text: string): Promise<Result<Idea>> {
  return call(gardenPath(projectId), {
    token,
    method: "POST",
    body: { text },
    refused: "frogly can't use that idea",
    tooLarge: `an idea has at most ${maxIdeaLength} characters`,
  });
}

export function meToo(token: Token, projectId: string, ideaId: string): Promise<Result<Idea>> {
  return call(`${gardenPath(projectId)}/${encodeURIComponent(ideaId)}/metoo`, { token, method: "POST" });
}

function projectPath(projectId: string): string {
  return `/api/projects/${encodeURIComponent(projectId)}`;
}

function requestsPath(projectId: string): string {
  return `${projectPath(projectId)}/requests`;
}

function gardenPath(projectId: string): string {
  return `${projectPath(projectId)}/ideas`;
}
