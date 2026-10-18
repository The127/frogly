import type { Failure, Result } from "./client";

// SessionEnds is what every page does when the login is gone, the same
// everywhere: the app shows the login page or says the account can't use
// frogly.
export interface SessionEnds {
  onLoggedOut(): void;
  onNotAllowed(): void;
}

// What a handler gives back. A text is shown: as the notice after a
// success, as the problem after a failure.
type Said = void | string | Promise<void | string>;

interface Handlers<T> {
  // what the action does, for its problem: "frogly couldn't <what>. Try
  // again in a moment."
  what: string;
  // what follows a success
  done(value: T): Said;
  // the thing is gone. Without this, the page's gone decides, and without
  // that it's a failure.
  notFound?(): Said;
  // someone changed the same thing at the same moment. Without this, it's
  // a failure, and trying again works.
  conflict?(): Said;
  // frogly or the network failed. Without this, the problem says to try
  // again.
  failed?(): Said;
}

// Busy is a group of actions that run one at a time, like a search and
// adding someone it found. Running one clears what all of them said. It
// keeps the actions of components that are gone too, which costs a few
// objects for as long as the page lives.
export class Busy {
  now = $state(false);
  #actions: Action[] = [];

  join(action: Action): void {
    this.#actions.push(action);
  }

  clear(): void {
    for (const action of this.#actions) {
      action.problem = "";
      action.notice = "";
    }
  }
}

// Action is one thing a person can do on a page, like planting an idea. It
// runs once at a time, and holds what the page shows about it.
export class Action {
  // what went wrong, for a role="alert" line
  problem = $state("");
  // what worked, for a polite live region
  notice = $state("");

  #session: SessionEnds;
  #gone: (() => void) | undefined;
  #busy: Busy;

  // gone is what the page does when the thing it's about isn't there any
  // more. busy is the group this action runs one at a time with.
  constructor(session: SessionEnds, gone?: () => void, busy = new Busy()) {
    this.#session = session;
    this.#gone = gone;
    this.#busy = busy;
    busy.join(this);
  }

  get busy(): boolean {
    return this.#busy.now;
  }

  async run<T>(work: () => Promise<Result<T>>, handlers: Handlers<T>): Promise<void> {
    if (this.#busy.now) {
      return;
    }

    this.#busy.now = true;
    this.#busy.clear();

    // busy until what follows is done too, like reloading a list, so a
    // second click can't run on what's about to change
    try {
      const result = await work();
      if (result.ok) {
        this.notice = (await handlers.done(result.value)) || "";
      } else {
        await this.#failed(result.failure, handlers);
      }
    } finally {
      this.#busy.now = false;
    }
  }

  async #failed<T>(failure: Failure, handlers: Handlers<T>): Promise<void> {
    const failed = `frogly couldn't ${handlers.what}. Try again in a moment.`;

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
          this.problem = (await handlers.notFound()) || "";
        } else if (this.#gone) {
          this.#gone();
        } else {
          this.problem = failed;
        }
        return;
      case "conflict":
        this.problem = handlers.conflict ? (await handlers.conflict()) || "" : failed;
        return;
      case "failed":
        this.problem = handlers.failed ? (await handlers.failed()) || "" : failed;
    }
  }
}

type Shown<T> = { kind: "loading" } | { kind: "loaded"; value: T } | { kind: "notFound" } | { kind: "failed" };

// Resource is what a page loads from frogly, like a project's garden.
export class Resource<T> {
  // raw, since what frogly sends is only ever replaced, never changed in
  // place, so it needs no deep proxy
  state = $state.raw<Shown<T>>({ kind: "loading" });
  // a load after the first one failed, so what it holds may be old
  stale = $state(false);

  #session: SessionEnds;
  #fetch: () => Promise<Result<T>>;
  #gone: (() => void) | undefined;
  #loads = 0;

  // gone is for what belongs to something else, like a project's requests:
  // when they aren't there, the whole page is gone
  constructor(session: SessionEnds, fetch: () => Promise<Result<T>>, gone?: () => void) {
    this.#session = session;
    this.#fetch = fetch;
    this.#gone = gone;
  }

  async load(): Promise<void> {
    const load = ++this.#loads;
    const result = await this.#fetch();
    // a later load started meanwhile and knows better
    if (load !== this.#loads) {
      return;
    }

    if (result.ok) {
      this.state = { kind: "loaded", value: result.value };
      this.stale = false;
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
        this.#gone?.();
        return;
    }

    if (this.state.kind === "loaded") {
      this.stale = true;
    } else {
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
