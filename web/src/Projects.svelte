<script lang="ts">
  import { onMount } from "svelte";

  import { type Project, createProject, fetchProjects } from "./api";

  // the token is asked for on every call, since it renews in the background
  let { accessToken }: { accessToken: () => Promise<string> } = $props();

  let projects = $state<Project[] | undefined>();
  let name = $state("");
  let problem = $state("");
  let saving = $state(false);
  let failed = $state(false);

  onMount(async () => {
    try {
      projects = await fetchProjects(await accessToken());
    } catch (error) {
      console.error(error);
      failed = true;
    }
  });

  async function create(event: SubmitEvent) {
    event.preventDefault();
    saving = true;
    problem = "";

    try {
      const created = await createProject(await accessToken(), name);
      if ("problem" in created) {
        problem = created.problem;
      } else {
        name = "";
        // the server sorts, so the list comes back in its order
        projects = await fetchProjects(await accessToken());
      }
    } catch (error) {
      console.error(error);
      problem = "frogly couldn't save the project. Try again in a moment.";
    } finally {
      saving = false;
    }
  }
</script>

<section aria-labelledby="projects-label">
  <p class="label" id="projects-label">your projects</p>

  {#if failed}
    <p class="quiet">frogly can't list your projects right now.</p>
  {:else if projects === undefined}
    <p class="quiet">…</p>
  {:else if projects.length === 0}
    <p class="quiet">No projects yet. Name your first one below.</p>
  {:else}
    <ul class="projects">
      {#each projects as project (project.id)}
        <li>{project.name}</li>
      {/each}
    </ul>
  {/if}

  <form onsubmit={create}>
    <input
      class="plant"
      bind:value={name}
      placeholder="name a project…"
      aria-label="Name of a new project"
      aria-describedby={problem ? "project-problem" : undefined}
      maxlength="100"
      disabled={saving}
    />
    {#if problem}
      <p class="problem" id="project-problem" role="alert">{problem}</p>
    {/if}
  </form>
</section>
