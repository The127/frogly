<script lang="ts">
  import { onMount } from "svelte";

  import { Action, Resource } from "../../lib/action.svelte";
  import { alive } from "../../lib/alive";
  import * as api from "../../lib/api";
  import { getLogin } from "../../lib/login";
  import { navigate } from "../../router";
  import Button from "../../ui/Button.svelte";
  import List from "../../ui/List.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Row from "../../ui/Row.svelte";
  import Rows from "../../ui/Rows.svelte";
  import Section from "../../ui/Section.svelte";

  const login = getLogin();
  const projects = new Resource(login, () => api.publicProjects(login.accessToken));
  // joining and asking share one action, so a person does one at a time
  const action = new Action(login, () => projects.load());
  onMount(() => projects.load());

  const shown = alive();

  async function open(projectId: string) {
    if (shown()) {
      await navigate("/projects/:id", { params: { id: projectId } });
    }
  }

  function join(project: api.PublicProject) {
    return action.run(() => api.join(login.accessToken, project.id), {
      what: `join ${project.name}`,
      done: () => open(project.id),
      // a 404 can mean closed, private or gone, and only the list knows
      // which of them still shows
      notFound: async () => {
        await projects.load();

        return `${project.name} can't be joined any more.`;
      },
    });
  }

  function ask(project: api.PublicProject) {
    return action.run(() => api.askToJoin(login.accessToken, project.id), {
      what: `ask to join ${project.name}`,
      done: async (waiting) => {
        // nothing waits for someone who is a member by now, like after a
        // member added them while this list was open
        if (!waiting) {
          await open(project.id);
          return;
        }

        projects.change((listed) => listed.map((other) => (other.id === project.id ? { ...other, asked: true } : other)));
      },
      notFound: async () => {
        await projects.load();

        return `${project.name} doesn't take requests any more.`;
      },
    });
  }
</script>

<Section label="public projects of others">
  <List resource={projects} failed="frogly can't list them right now." empty="Nobody else has a public project yet.">
    {#snippet children(listed)}
      <!-- not links, a garden opens to its members -->
      <Rows>
        {#each listed as project (project.id)}
          <Row serif>
            {project.name}
            {#snippet actions()}
              {#if project.joining === "open"}
                <Button busy={action.busy} onclick={() => join(project)}>join <span class="hidden">{project.name}</span></Button>
              {:else if project.joining === "request" && project.asked}
                <span class="quiet small">you asked, a member will answer</span>
              {:else if project.joining === "request"}
                <Button busy={action.busy} onclick={() => ask(project)}>ask to join <span class="hidden">{project.name}</span></Button>
              {:else}
                <span class="quiet small">a member adds people</span>
              {/if}
            {/snippet}
          </Row>
        {/each}
      </Rows>
    {/snippet}
  </List>
  <Problem text={action.problem} />
</Section>
