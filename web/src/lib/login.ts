import { createContext } from "svelte";

import type { SessionEnds } from "./action.svelte";
import type { Token } from "./client";

// What every page needs from the login. The router renders pages without
// props, so App hands this down as a context.
export interface Login extends SessionEnds {
  name: string;
  // asked for on every call, since the token renews in the background
  accessToken: Token;
}

export const [getLogin, setLogin] = createContext<Login>();
