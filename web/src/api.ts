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
