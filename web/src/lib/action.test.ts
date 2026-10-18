import { describe, expect, it, vi } from "vitest";

import { Action, Busy, Resource } from "./action.svelte";
import type { Failure, Result } from "./client";

function session() {
  return { onLoggedOut: vi.fn(), onNotAllowed: vi.fn() };
}

function ok<T>(value: T): () => Promise<Result<T>> {
  return async () => ({ ok: true, value });
}

function failing(failure: Failure): () => Promise<Result<never>> {
  return async () => ({ ok: false, failure });
}

describe("Action", () => {
  it("hands the value on and shows what it says", async () => {
    // arrange
    const action = new Action(session(), vi.fn());
    const done = vi.fn(() => "Ana is a member now.");

    // act
    await action.run(ok("u-2"), { what: "add Ana", done });

    // assert
    expect(done).toHaveBeenCalledWith("u-2");
    expect(action.notice).toBe("Ana is a member now.");
    expect(action.problem).toBe("");
    expect(action.busy).toBe(false);
  });

  it("is busy until the work and what follows it are done", async () => {
    // arrange
    const action = new Action(session(), vi.fn());
    let busyWhileDone = false;

    // act
    await action.run(ok(1), {
      what: "plant",
      done: async () => {
        busyWhileDone = action.busy;
      },
    });

    // assert
    expect(busyWhileDone).toBe(true);
    expect(action.busy).toBe(false);
  });

  it("runs once at a time", async () => {
    // arrange
    const action = new Action(session(), vi.fn());
    let finish: (result: Result<number>) => void = () => {};
    const work = vi.fn(() => new Promise<Result<number>>((resolve) => (finish = resolve)));

    // act
    const first = action.run(work, { what: "plant", done: () => {} });
    await action.run(work, { what: "plant", done: () => {} });
    finish({ ok: true, value: 1 });
    await first;

    // assert
    expect(work).toHaveBeenCalledTimes(1);
  });

  it("says what failed and to try again", async () => {
    // arrange
    const action = new Action(session(), vi.fn());

    // act
    await action.run(failing({ kind: "failed" }), { what: "plant the idea", done: () => {} });

    // assert
    expect(action.problem).toBe("frogly couldn't plant the idea. Try again in a moment.");
  });

  it("counts a conflict as a failure unless the action knows better", async () => {
    // arrange
    const plain = new Action(session(), vi.fn());
    const knowing = new Action(session(), vi.fn());
    const conflict = vi.fn();

    // act
    await plain.run(failing({ kind: "conflict" }), { what: "add Ana", done: () => {} });
    await knowing.run(failing({ kind: "conflict" }), { what: "answer Ana", done: () => {}, conflict });

    // assert
    expect(plain.problem).toBe("frogly couldn't add Ana. Try again in a moment.");
    expect(conflict).toHaveBeenCalled();
    expect(knowing.problem).toBe("");
  });

  it("shows frogly's reason when it refuses", async () => {
    // arrange
    const action = new Action(session(), vi.fn());

    // act
    await action.run(failing({ kind: "refused", problem: "an idea needs a text" }), { what: "plant", done: () => {} });

    // assert
    expect(action.problem).toBe("an idea needs a text");
  });

  it("tells the page it's gone when something isn't there, unless the action knows better", async () => {
    // arrange
    const gone = vi.fn();
    const notFound = vi.fn();
    const plain = new Action(session(), gone);
    const knowing = new Action(session(), gone);

    // act
    await plain.run(failing({ kind: "notFound" }), { what: "plant", done: () => {} });
    await knowing.run(failing({ kind: "notFound" }), { what: "join", done: () => {}, notFound });

    // assert
    expect(gone).toHaveBeenCalledTimes(1);
    expect(notFound).toHaveBeenCalledTimes(1);
  });

  it.each([
    { kind: "loggedOut" as const, called: "onLoggedOut" as const },
    { kind: "notAllowed" as const, called: "onNotAllowed" as const },
  ])("hands a $kind login to the session", async ({ kind, called }) => {
    // arrange
    const ends = session();
    const action = new Action(ends, vi.fn());

    // act
    await action.run(failing({ kind }), { what: "plant", done: () => {} });

    // assert
    expect(ends[called]).toHaveBeenCalled();
    expect(action.problem).toBe("");
  });

  it("forgets the last problem and notice when it runs again", async () => {
    // arrange
    const action = new Action(session(), vi.fn());
    await action.run(failing({ kind: "failed" }), { what: "plant", done: () => {} });

    // act
    await action.run(ok(1), { what: "plant", done: () => {} });

    // assert
    expect(action.problem).toBe("");
    expect(action.notice).toBe("");
  });
});

describe("Action, more", () => {
  it("is not busy any more when the work throws", async () => {
    // arrange
    const action = new Action(session(), vi.fn());

    // act
    const run = action.run(
      async () => {
        throw new Error("a bug");
      },
      { what: "plant", done: () => {} },
    );

    // assert
    await expect(run).rejects.toThrow("a bug");
    expect(action.busy).toBe(false);
  });

  it("shows no notice for an empty text", async () => {
    // arrange
    const action = new Action(session(), vi.fn());

    // act
    await action.run(ok(1), { what: "plant", done: () => "" });

    // assert
    expect(action.notice).toBe("");
  });

  it("lets a failure go on its own way", async () => {
    // arrange
    const action = new Action(session(), vi.fn());
    const failed = vi.fn();

    // act
    await action.run(failing({ kind: "failed" }), { what: "count the members", done: () => {}, failed });

    // assert
    expect(failed).toHaveBeenCalled();
    expect(action.problem).toBe("");
  });

  it("waits for another action that shares its busy flag", async () => {
    // arrange
    const busy = new Busy();
    const search = new Action(session(), vi.fn(), busy);
    const add = new Action(session(), vi.fn(), busy);
    let finish: (result: Result<number>) => void = () => {};
    const work = vi.fn(async () => ({ ok: true, value: 1 }) as Result<number>);

    // act
    const searching = search.run(() => new Promise<Result<number>>((resolve) => (finish = resolve)), { what: "search", done: () => {} });
    const addBusy = add.busy;
    await add.run(work, { what: "add", done: () => {} });
    finish({ ok: true, value: 1 });
    await searching;

    // assert
    expect(addBusy).toBe(true);
    expect(work).not.toHaveBeenCalled();
    expect(add.busy).toBe(false);
  });
});

describe("Resource", () => {
  it("starts loading and then holds what it loaded", async () => {
    // arrange
    const resource = new Resource(session(), ok(["a garden"]));
    const before = resource.state.kind;

    // act
    await resource.load();

    // assert
    expect(before).toBe("loading");
    expect(resource.state).toEqual({ kind: "loaded", value: ["a garden"] });
  });

  it.each([
    { failure: { kind: "notFound" } as Failure, state: "notFound" },
    { failure: { kind: "failed" } as Failure, state: "failed" },
    { failure: { kind: "conflict" } as Failure, state: "failed" },
  ])("is $state after a $failure.kind", async ({ failure, state }) => {
    // arrange
    const resource = new Resource(session(), failing(failure));

    // act
    await resource.load();

    // assert
    expect(resource.state.kind).toBe(state);
  });

  it("hands a lost login to the session", async () => {
    // arrange
    const ends = session();
    const resource = new Resource(ends, failing({ kind: "loggedOut" }));

    // act
    await resource.load();

    // assert
    expect(ends.onLoggedOut).toHaveBeenCalled();
  });

  it("changes what it holds", async () => {
    // arrange
    const resource = new Resource(session(), ok([1, 2]));
    await resource.load();

    // act
    resource.change((numbers) => [0, ...numbers]);

    // assert
    expect(resource.state).toEqual({ kind: "loaded", value: [0, 1, 2] });
  });

  it("says it's gone", () => {
    // arrange
    const resource = new Resource(session(), ok(1));

    // act
    resource.gone();

    // assert
    expect(resource.state.kind).toBe("notFound");
  });
});

describe("Resource, loaded again", () => {
  it("keeps what it had when loading again fails, and says it's stale", async () => {
    // arrange
    let answer: Result<number[]> = { ok: true, value: [1] };
    const resource = new Resource(session(), async () => answer);
    await resource.load();
    answer = { ok: false, failure: { kind: "failed" } };

    // act
    await resource.load();

    // assert
    expect(resource.state).toEqual({ kind: "loaded", value: [1] });
    expect(resource.stale).toBe(true);
  });

  it("is fresh again after a load that works", async () => {
    // arrange
    let answer: Result<number[]> = { ok: false, failure: { kind: "failed" } };
    const resource = new Resource(session(), async () => answer);
    await resource.load();
    answer = { ok: true, value: [2] };

    // act
    await resource.load();

    // assert
    expect(resource.state).toEqual({ kind: "loaded", value: [2] });
    expect(resource.stale).toBe(false);
  });

  it("holds the latest load when an earlier one finishes last", async () => {
    // arrange
    const finishes: ((result: Result<string>) => void)[] = [];
    const resource = new Resource(session(), () => new Promise<Result<string>>((resolve) => finishes.push(resolve)));

    // act
    const first = resource.load();
    const second = resource.load();
    finishes[1]({ ok: true, value: "new" });
    await second;
    finishes[0]({ ok: true, value: "old" });
    await first;

    // assert
    expect(resource.state).toEqual({ kind: "loaded", value: "new" });
  });

  it("tells the page when what it loads belongs to something gone", async () => {
    // arrange
    const gone = vi.fn();
    const resource = new Resource(session(), failing({ kind: "notFound" }), gone);

    // act
    await resource.load();

    // assert
    expect(gone).toHaveBeenCalled();
  });
});

describe("Action, in a group", () => {
  it("clears what every action of its group said when one runs", async () => {
    // arrange
    const busy = new Busy();
    const search = new Action(session(), vi.fn(), busy);
    const add = new Action(session(), vi.fn(), busy);
    await add.run(failing({ kind: "failed" }), { what: "add Ana", done: () => {} });
    await add.run(ok(1), { what: "add Ana", done: () => "Ana is a member now." });

    // act
    await search.run(failing({ kind: "failed" }), { what: "search", done: () => {} });

    // assert
    expect(add.notice).toBe("");
    expect(add.problem).toBe("");
    expect(search.problem).toBe("frogly couldn't search. Try again in a moment.");
  });
});

describe("Action, failing its own way", () => {
  it.each([
    { failure: { kind: "notFound" } as Failure, handler: "notFound" as const },
    { failure: { kind: "conflict" } as Failure, handler: "conflict" as const },
    { failure: { kind: "failed" } as Failure, handler: "failed" as const },
  ])("shows the problem its $handler handler gives", async ({ failure, handler }) => {
    // arrange
    const action = new Action(session());

    // act
    await action.run(failing(failure), { what: "answer Ana", done: () => {}, [handler]: () => "Ana's request was answered already." });

    // assert
    expect(action.problem).toBe("Ana's request was answered already.");
  });

  it("counts a missing thing as a failure when the page has no other answer for it", async () => {
    // arrange
    const action = new Action(session());

    // act
    await action.run(failing({ kind: "notFound" }), { what: "save the project", done: () => {} });

    // assert
    expect(action.problem).toBe("frogly couldn't save the project. Try again in a moment.");
  });
});
