<script lang="ts">
  import Garden from "./Garden.svelte";
  import { route } from "./router";

  const projectId = $derived((route.params as { id?: string }).id ?? "");
</script>

<!-- the router keeps the page when only the ID changes, like on back and
     forward. A new garden for each project keeps nothing of the last one,
     not even a planting that finishes late. -->
{#key projectId}
  <Garden {projectId} />
{/key}
