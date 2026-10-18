import { createRouter } from "sv-router";

// Router renders the routes createRouter registered. Taking it from here
// makes sure they're registered before it renders.
export { Router } from "sv-router";

// The pages are imported lazily, since they import p and route from here.
// Imported up front, a page that loads first would run createRouter before
// the pages exist.
export const { p, navigate, route } = createRouter({
  "/": () => import("./pages/Home.svelte"),
  "/projects/new": () => import("./pages/NewProject.svelte"),
  "/projects/:id": () => import("./GardenPage.svelte"),
  "/projects/:id/settings": () => import("./SettingsPage.svelte"),
  "*": () => import("./Lost.svelte"),
});
