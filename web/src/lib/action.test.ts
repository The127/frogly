import { describe, expect, it, vi } from "vitest";

import { Action, Resource } from "./action.svelte";
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
