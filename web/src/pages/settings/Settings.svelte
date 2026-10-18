<script lang="ts">
  import { onMount } from "svelte";

  import { Busy, Resource } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { projectGone } from "../../lib/format";
  import { getLogin } from "../../lib/login";
  import { p } from "../../router";
  import Back from "../../ui/Back.svelte";
  import Loaded from "../../ui/Loaded.svelte";
  import Page from "../../ui/Page.svelte";
  import AddPeople from "./AddPeople.svelte";
  import Joining from "./Joining.svelte";
  import Leave from "./Leave.svelte";
  import Requests from "./Requests.svelte";
  import Visibility from "./Visibility.svelte";

  let { projectId }: { projectId: string } = $props();

  const login = getLogin();
  const settings = new Resource(login, () => api.project(login.accessToken, projectId));
  const gone = () => settings.gone();
  // the requests belong to the project, so when they're not found, it's gone
  const requests = new Resource(login, () => api.requests(login.accessToken, projectId), gone);
  // who sees the project and who can join it are saved one at a time, and
  // saving either clears what went wrong with the other
  const saving = new Busy();

  // both at once, so the requests don't push the page down later
  onMount(() => Promise.all([settings.load(), requests.load()]));

  function changed(project: api.Project) {
    settings.change((shown) => ({ ...shown, ...project }));
    // frogly drops the requests of a project that doesn't take them any more
    if (project.joining !== "request") {
      requests.change(() => []);
    }
  }
</script>

<Back href={p("/projects/:id", { params: { id: projectId } })} text="the garden" />

<Loaded
  resource={settings}
  title="Settings"
  notFound={projectGone}
  failed="These settings can't be shown right now."
>
  {#snippet children(project)}
    <Page title={`Settings for ${project.name}`} heading={`Settings for ${project.name}`}>
      <Visibility {project} busy={saving} {changed} {gone} />
      {#if project.visibility === "public"}
        <Joining {project} busy={saving} {changed} {gone} />
        <Requests {projectId} {requests} {gone} />
      {/if}
      <AddPeople {projectId} {gone} />
      <Leave {project} {gone} />
    </Page>
  {/snippet}
</Loaded>
