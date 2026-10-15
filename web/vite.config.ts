import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    strictPort: true,
    // the dev server serves the frontend, the Go server answers the API,
    // and the browser sees one origin
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
