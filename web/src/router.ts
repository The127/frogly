import { createRouter } from "sv-router";

// Router renders the routes createRouter registered. Taking it from here
// makes sure they're registered before it renders.
export { Router } from "sv-router";

import GardenPage from "./GardenPage.svelte";
import Home from "./Home.svelte";
import Lost from "./Lost.svelte";

export const { p, route } = createRouter({
  "/": Home,
  "/projects/:id": GardenPage,
  "*": Lost,
});
