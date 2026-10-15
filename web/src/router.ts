import { createRouter } from "sv-router";

import Garden from "./Garden.svelte";
import Home from "./Home.svelte";
import Lost from "./Lost.svelte";

export const { p, route } = createRouter({
  "/": Home,
  "/projects/:id": Garden,
  "*": Lost,
});
