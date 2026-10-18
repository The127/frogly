import type { Failure, Result } from "./client";

// SessionEnds is what every page does when the login is gone, the same
// everywhere: the app shows the login page or says the account can't use
// frogly.
export interface SessionEnds {
  onLoggedOut(): void;
  onNotAllowed(): void;
}

export interface Handlers<T> {
  // what the action does, for its problem: "frogly couldn't <what>. Try
  // again in a moment."
  what: string;
  // what follows a success. A text it gives is shown as the action's
  // notice.
  done(value: T): void | string | Promise<void | string>;
  // the thing is gone. Without this, the whole page is.
  notFound?(): void | Promise<void>;
  // someone changed the same thing at the same moment. Without this, it's
  // a failure, and trying again works.
  conflict?(): void | Promise<void>;
}

// Action is one thing a person can do on a page, like planting an idea. It
// runs once at a time, and holds what the page shows about it.
export class Action {
  busy = $state(false);
  // what went wrong, for a role="alert" line
  problem = $state("");
  // what worked, for a polite live region
  notice = $state("");

  #session: SessionEnds;
  #gone: () => void;

  // gone is what the page does when the thing it's about isn't there any
  // more
  constructor(session: SessionEnds, gone: () => void) {
    this.#session = session;
    this.#gone = gone;
  }

  async run<T>(work: () => Promise<Result<T>>, handlers: Handlers<T>): Promise<void> {
    if (this.busy) {
      return;
    }

    this.busy = true;
    this.problem = "";
    this.notice = "";

    // busy until what follows is done too, like reloading a list, so a
    // second click can't run on what's about to change
    try {
      const result = await work();
      if (result.ok) {
        const notice = await handlers.done(result.value);
        if (notice) {
          this.notice = notice;
        }
      } else {
        await this.#failed(result.failure, handlers);
      }
    } finally {
      this.busy = false;
    }
  }

  async #failed<T>(failure: Failure, handlers: Handlers<T>): Promise<void> {
    switch (failure.kind) {
      case "loggedOut":
        this.#session.onLoggedOut();
        return;
      case "notAllowed":
        this.#session.onNotAllowed();
        return;
      case "refused":
        this.problem = failure.problem;
        return;
      case "notFound":
        if (handlers.notFound) {
          await handlers.notFound();
        } else {
          this.#gone();
        }
        return;
      case "conflict":
        if (handlers.conflict) {
          await handlers.conflict();
          return;
        }
    }

    this.problem = `frogly couldn't ${handlers.what}. Try again in a moment.`;
  }
}

export type Loaded<T> = { kind: "loading" } | { kind: "loaded"; value: T } | { kind: "notFound" } | { kind: "failed" };

// Resource is what a page loads from frogly, like a project's garden.
export class Resource<T> {
  state = $state<Loaded<T>>({ kind: "loading" });

  #session: SessionEnds;
  #fetch: () => Promise<Result<T>>;

  constructor(session: SessionEnds, fetch: () => Promise<Result<T>>) {
    this.#session = session;
    this.#fetch = fetch;
  }

  async load(): Promise<void> {
    const result = await this.#fetch();
    if (result.ok) {
      this.state = { kind: "loaded", value: result.value };
      return;
    }

    switch (result.failure.kind) {
      case "loggedOut":
        this.#session.onLoggedOut();
        return;
      case "notAllowed":
        this.#session.onNotAllowed();
        return;
      case "notFound":
        this.state = { kind: "notFound" };
        return;
      default:
        this.state = { kind: "failed" };
    }
  }

  // change keeps what an action did, without loading everything again
  change(update: (value: T) => T): void {
    if (this.state.kind === "loaded") {
      this.state = { kind: "loaded", value: update(this.state.value) };
    }
  }

  // gone is for an action that finds the thing isn't there any more
  gone(): void {
    this.state = { kind: "notFound" };
  }
}
