// Failure is how a call to frogly went wrong, in the terms a page acts on.
export type Failure =
  // the login ended, and only logging in again helps
  | { kind: "loggedOut" }
  // the login works, but it doesn't give the person frogly's role
  | { kind: "notAllowed" }
  // the thing isn't there, or the person may not see it. frogly doesn't
  // say which.
  | { kind: "notFound" }
  // someone changed the same thing at the same moment
  | { kind: "conflict" }
  // frogly refused what the person sent, for a reason they can act on
  | { kind: "refused"; problem: string }
  // frogly or the network failed, and trying again later may work
  | { kind: "failed" };

export type Result<T> = { ok: true; value: T } | { ok: false; failure: Failure };

// LoggedOut is what a token getter throws when the login ended, before
// anything is sent.
export class LoggedOut extends Error {
  constructor() {
    super("the login ended");
    this.name = "LoggedOut";
  }
}

export type Token = () => Promise<string>;

interface Options {
  token?: Token;
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  // the reason to give when frogly refuses without one. A call without it
  // sends only what frogly made itself, so a refusal is frogly's problem.
  refused?: string;
  // the reason to give when what was sent is far too long for frogly to
  // read, which it answers without a reason
  tooLarge?: string;
}

// call is the only way the frontend talks to frogly. It never throws: every
// answer becomes a Result.
export async function call<T>(path: string, options: Options = {}): Promise<Result<T>> {
  const { token, method = "GET", body, refused, tooLarge } = options;

  let response: Response;
  try {
    const headers: Record<string, string> = {};
    if (token) {
      headers.Authorization = `Bearer ${await token()}`;
    }

    const init: RequestInit = { method, headers };
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }

    response = await fetch(path, init);
  } catch (error) {
    if (error instanceof LoggedOut) {
      return failed({ kind: "loggedOut" });
    }

    console.error(`calling ${method} ${path} failed`, error);

    return failed({ kind: "failed" });
  }

  if (response.ok) {
    // a 204 has no body to read
    if (response.status === 204) {
      return { ok: true, value: undefined as T };
    }

    try {
      return { ok: true, value: (await response.json()) as T };
    } catch (error) {
      // something between the browser and frogly answered, like a proxy
      console.error(`frogly's answer to ${method} ${path} isn't JSON`, error);

      return failed({ kind: "failed" });
    }
  }

  if (response.status === 413 && tooLarge !== undefined) {
    return failed({ kind: "refused", problem: tooLarge });
  }

  return failed(await failure(response, method, path, refused));
}

async function failure(response: Response, method: string, path: string, refused: string | undefined): Promise<Failure> {
  switch (response.status) {
    case 401:
      return { kind: "loggedOut" };
    case 403:
      return { kind: "notAllowed" };
    case 404:
      return { kind: "notFound" };
    case 409:
      return { kind: "conflict" };
  }

  if (refused !== undefined && response.status === 400) {
    return { kind: "refused", problem: await reason(response, refused) };
  }

  console.error(`frogly answered ${response.status} to ${method} ${path}`);

  return { kind: "failed" };
}

// the reason in frogly's problem, or fallback when it has none
async function reason(response: Response, fallback: string): Promise<string> {
  const problem = (await response.json().catch(() => ({}))) as { detail?: string };

  return problem.detail ?? fallback;
}

function failed<T>(failure: Failure): Result<T> {
  return { ok: false, failure };
}

// both runs two calls side by side, for a page that needs both, and gives
// the first failure if either fails
export async function both<A, B>(a: Promise<Result<A>>, b: Promise<Result<B>>): Promise<Result<[A, B]>> {
  const [first, second] = await Promise.all([a, b]);
  if (!first.ok) {
    return first;
  }

  if (!second.ok) {
    return second;
  }

  return { ok: true, value: [first.value, second.value] };
}

// map changes the value of a call that worked, and keeps a failure as it is
export function map<A, B>(result: Result<A>, change: (value: A) => B): Result<B> {
  return result.ok ? { ok: true, value: change(result.value) } : result;
}
