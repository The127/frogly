<script lang="ts">
  import { onMount } from "svelte";

  import type { PublicProject } from "./api";
  import { getLogin } from "./login";
  import { loadPublicProjects } from "./projects";

  const { accessToken, onLoggedOut, onNotAllowed } = getLogin();

  let projects = $state<PublicProject[] | undefined>();
  let listFailed = $state(false);

  onMount(async () => {
    const loaded = await loadPublicProjects(accessToken);
    switch (loaded.kind) {
      case "listed":
        projects = loaded.projects;
        break;
      case "failed":
        listFailed = true;
        break;
      case "loggedOut":
        onLoggedOut();
        break;
      case "notAllowed":
        onNotAllowed();
    }
  });
</script>

<section aria-labelledby="public-label">
  <p class="label" id="public-label">public projects of others</p>

  {#if listFailed}
    <p class="quiet">frogly can't list them right now. Reload the page in a moment.</p>
  {:else if projects === undefined}
    <p class="quiet">…</p>
  {:else if projects.length === 0}
    <p class="quiet">Nobody else has a public project yet.</p>
  {:else}
    <!-- names only, a garden opens to its members -->
    <ul class="projects">
      {#each projects as project (project.id)}
        <li>{project.name}</li>
      {/each}
    </ul>
  {/if}
</section>
