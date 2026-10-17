<script lang="ts">
  import { route } from "./router";
  import Settings from "./Settings.svelte";

  const projectId = $derived((route.params as { id?: string }).id ?? "");
</script>

<!-- like GardenPage, fresh settings for each project -->
{#key projectId}
  <Settings {projectId} />
{/key}
