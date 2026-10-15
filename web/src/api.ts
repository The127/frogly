export interface Config {
  issuer: string;
  clientId: string;
}

export interface Me {
  name: string;
}

export async function fetchConfig(): Promise<Config> {
  const response = await fetch("/api/config");
  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked where to log in`);
  }

  return (await response.json()) as Config;
}

// null means frogly refused the token, so the person has to log in again.
// Any other failure is frogly's problem, not the person's.
export async function fetchMe(accessToken: string): Promise<Me | null> {
  const response = await fetch("/api/me", {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (response.status === 401) {
    return null;
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked who is logged in`);
  }

  return (await response.json()) as Me;
}

export interface Project {
  id: string;
  name: string;
}

export async function fetchProjects(accessToken: string): Promise<Project[]> {
  const response = await fetch("/api/projects", {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked for your projects`);
  }

  return (await response.json()) as Project[];
}

// Either the new project, or frogly's reason for refusing the name, which
// the person can act on. Any other failure is frogly's problem.
export type Created = { project: Project } | { problem: string };

export async function createProject(accessToken: string, name: string): Promise<Created> {
  const response = await fetch("/api/projects", {
    method: "POST",
    headers: { Authorization: `Bearer ${accessToken}`, "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (response.status === 400) {
    const problem = (await response.json()) as { detail?: string };

    return { problem: problem.detail ?? "frogly can't use that name" };
  }

  if (!response.ok) {
    throw new Error(`frogly answered ${response.status} when asked to create a project`);
  }

  return { project: (await response.json()) as Project };
}
