<script lang="ts">
  import type { Component } from "svelte";

  import { route } from "../router";

  let { page: ProjectPage }: { page: Component<{ projectId: string }> } = $props();

  const projectId = $derived((route.params as { id?: string }).id ?? "");
</script>

<!-- the router keeps the page when only the ID changes, like on back and
     forward. A new page for each project keeps nothing of the last one,
     not even an action that finishes late. -->
{#key projectId}
  <ProjectPage {projectId} />
{/key}
