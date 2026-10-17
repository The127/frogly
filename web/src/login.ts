import { createContext } from "svelte";

// What every page needs from the login. The router renders pages without
// props, so App hands this down as a context.
export interface Login {
  name: string;
  // asked for on every call, since the token renews in the background
  accessToken: () => Promise<string>;
  onLoggedOut: () => void;
  // the login works, but frogly's role was taken away
  onNotAllowed: () => void;
}

export const [getLogin, setLogin] = createContext<Login>();
