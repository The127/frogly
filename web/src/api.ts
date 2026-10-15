export interface Config {
  issuer: string;
  clientId: string;
}

export interface Me {
  name: string;
}

export interface Project {
  id: string;
  name: string;
}

export interface Idea {
  id: string;
  text: string;
  plantedBy: string;
  // an RFC 3339 time in UTC
  plantedAt: string;
}

// LoggedOut means the login ended, and trying again won't help until the
// person logs in again.
export class LoggedOut extends Error {
  constructor() {
    super("the login ended");
    this.name = "LoggedOut";
  }
}

export async function fetchConfig(): Promise<Config> {
  const response = await fetch("/api/config");
  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked where to log in`);
  }

  return (await response.json()) as Config;
}

// NotFound means the project doesn't exist or the person isn't a member of
// it. frogly doesn't say which.
export class NotFound extends Error {
  constructor() {
    super("the project isn't there");
    this.name = "NotFound";
  }
}

// Every call that needs the login goes through here, so a refused token
// always ends up as LoggedOut.
async function authorized(path: string, accessToken: string, init: RequestInit = {}): Promise<Response> {
  const response = await fetch(path, {
    ...init,
    headers: { Authorization: `Bearer ${accessToken}`, ...init.headers },
  });
  if (response.status === 401) {
    throw new LoggedOut();
  }

  return response;
}

// null means frogly refused the token, so the person has to log in again.
// Any other failure is frogly's problem, not the person's.
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

// Either the new project, or frogly's reason for refusing the name, which
// the person can act on. Any other failure is frogly's problem.
export type Created = { project: Project } | { problem: string };

export async function createProject(accessToken: string, name: string): Promise<Created> {
  const response = await authorized("/api/projects", accessToken, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (response.status === 400) {
    return { problem: await refusal(response, "frogly can't use that name") };
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

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to plant an idea`);
  }

  return { idea: (await response.json()) as Idea };
}
