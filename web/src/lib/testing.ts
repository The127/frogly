import { vi } from "vitest";

// what the tests of the client and the endpoints share: a login that gives
// a token, and frogly answering every call the same way

export const token = async () => "an-access-token";

export function respondWith(status: number, body?: unknown): void {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(body === undefined ? null : JSON.stringify(body), { status })),
  );
}
