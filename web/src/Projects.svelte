<script lang="ts">
  import { onMount } from "svelte";

  import type { Project, Visibility } from "./api";
  import { getLogin } from "./login";
  import { addProject, loadProjects } from "./projects";
  import { p } from "./router";

  const { accessToken, onLoggedOut, onNotAllowed } = getLogin();

  let projects = $state<Project[] | undefined>();
  let listFailed = $state(false);
  let name = $state("");
  // nothing is chosen in advance, the person decides who sees a project
  let visibility = $state<Visibility | undefined>();
  let problem = $state("");
  let saving = $state(false);

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

  async function create(event: SubmitEvent) {
    event.preventDefault();
    if (saving) {
      return;
    }

    saving = true;
    problem = "";

    const added = await addProject(accessToken, name, visibility);
    saving = false;

    switch (added.kind) {
      case "saved":
        name = "";
        visibility = undefined;
        projects = added.projects;
        listFailed = false;
        break;
      case "savedButNotListed":
        name = "";
        visibility = undefined;
        listFailed = true;
        break;
      case "refused":
        problem = added.problem;
        break;
      case "notSaved":
        problem = "frogly couldn't save the project. Try again in a moment.";
        break;
      case "loggedOut":
        onLoggedOut();
        break;
      case "notAllowed":
        onNotAllowed();
    }
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

  <form onsubmit={create}>
    <label class="hidden" for="project-name">Name of a new project</label>
    <!-- readonly while saving, since a disabled field would lose the focus -->
    <input
      class="plant"
      id="project-name"
      bind:value={name}
      placeholder="name a project…"
      readonly={saving}
      aria-busy={saving}
      aria-invalid={problem ? "true" : undefined}
      aria-describedby="project-problem"
    />
    <!-- not disabled while saving, a disabled radio would lose the focus.
         create() already refuses a second save. -->
    <fieldset class="choice" aria-describedby="project-problem">
      <legend class="hidden">Who sees the new project</legend>
      <label><input type="radio" name="visibility" value="private" bind:group={visibility} /> private, only its members see it</label>
      <label><input type="radio" name="visibility" value="public" bind:group={visibility} /> public, everyone in frogly sees its name</label>
    </fieldset>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" id="project-problem" role="alert">{problem}</p>
  </form>
</section>
