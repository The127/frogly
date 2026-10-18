<script lang="ts">
  import { onMount } from "svelte";

  import { Resource } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { getLogin } from "../../lib/login";
  import { navigate, p } from "../../router";
  import List from "../../ui/List.svelte";
  import Row from "../../ui/Row.svelte";
  import Rows from "../../ui/Rows.svelte";
  import Section from "../../ui/Section.svelte";
  import TextField from "../../ui/TextField.svelte";

  const login = getLogin();
  const projects = new Resource(login, () => api.projects(login.accessToken));
  onMount(() => projects.load());

  let name = $state("");

  // the name goes on to the page for a new project, where the rest is
  // chosen
  function start(event: SubmitEvent) {
    event.preventDefault();
    void navigate("/projects/new", { search: { name } });
  }
</script>

<Section label="your projects" id="projects-label">
  <List resource={projects} failed="frogly can't list your projects right now. Reload the page in a moment." empty="No projects yet. Name your first one below.">
    {#snippet children(listed)}
      <Rows>
        {#each listed as project (project.id)}
          <Row serif><a class="plain" href={p("/projects/:id", { params: { id: project.id } })}>{project.name}</a></Row>
        {/each}
      </Rows>
    {/snippet}
  </List>

  <form onsubmit={start}>
    <TextField id="project-name" label="Name of a new project" bind:value={name} placeholder="name a project…" />
  </form>
</Section>
