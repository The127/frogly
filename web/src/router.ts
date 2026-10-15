import { createRouter } from "sv-router";

import GardenPage from "./GardenPage.svelte";
import Home from "./Home.svelte";
import Lost from "./Lost.svelte";

export const { p, route } = createRouter({
  "/": Home,
  "/projects/:id": GardenPage,
  "*": Lost,
});
