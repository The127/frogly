<script lang="ts">
  import { route } from "./router";
  import Settings from "./Settings.svelte";

  const projectId = $derived((route.params as { id?: string }).id ?? "");
</script>

<!-- the router keeps the page when only the ID changes, so a change that
     finishes late mustn't land on the next project's settings -->
{#key projectId}
  <Settings {projectId} />
{/key}
