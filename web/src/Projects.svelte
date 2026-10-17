<script lang="ts">
  import { onMount } from "svelte";

  import type { Project } from "./api";
  import { getLogin } from "./login";
  import { loadProjects } from "./projects";
  import { navigate, p } from "./router";

  const { accessToken, onLoggedOut, onNotAllowed } = getLogin();

  let projects = $state<Project[] | undefined>();
  let listFailed = $state(false);
  let name = $state("");

  onMount(async () => {
    const loaded = await loadProjects(accessToken);
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

  function startProject(event: SubmitEvent) {
    event.preventDefault();
    void navigate("/projects/new", { search: { name } });
  }
</script>

<section aria-labelledby="projects-label">
  <p class="label" id="projects-label">your projects</p>

  {#if listFailed}
    <p class="quiet">frogly can't list your projects right now. Reload the page in a moment.</p>
  {:else if projects === undefined}
    <p class="quiet">…</p>
  {:else if projects.length === 0}
    <p class="quiet">No projects yet. Name your first one below.</p>
  {:else}
    <ul class="projects">
      {#each projects as project (project.id)}
        <li><a href={p("/projects/:id", { params: { id: project.id } })}>{project.name}</a></li>
      {/each}
    </ul>
  {/if}

  <form onsubmit={startProject}>
    <label class="hidden" for="project-name">Name of a new project</label>
    <input class="plant" id="project-name" bind:value={name} placeholder="name a project…" />
  </form>
</section>
