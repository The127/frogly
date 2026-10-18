<script lang="ts">
  import { onMount } from "svelte";

  import { Action, Resource } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { both } from "../../lib/client";
  import { peopleBehind, plantedOn } from "../../lib/format";
  import { getLogin } from "../../lib/login";
  import { p } from "../../router";
  import Back from "../../ui/Back.svelte";
  import Button from "../../ui/Button.svelte";
  import Loaded from "../../ui/Loaded.svelte";
  import Page from "../../ui/Page.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Row from "../../ui/Row.svelte";
  import Rows from "../../ui/Rows.svelte";
  import Section from "../../ui/Section.svelte";
  import TextField from "../../ui/TextField.svelte";

  let { projectId }: { projectId: string } = $props();

  const login = getLogin();
  const garden = new Resource(login, async () => {
    const loaded = await both(api.project(login.accessToken, projectId), api.garden(login.accessToken, projectId));

    return loaded.ok ? { ok: true as const, value: { project: loaded.value[0], ideas: loaded.value[1] } } : loaded;
  });
  const planting = new Action(login, () => garden.gone());
  const sayingMeToo = new Action(login, () => garden.gone());
  onMount(() => garden.load());

  let text = $state("");

  function plant(event: SubmitEvent) {
    event.preventDefault();

    return planting.run(() => api.plant(login.accessToken, projectId, text), {
      what: "plant the idea",
      done: (idea) => {
        text = "";
        garden.change((shown) => ({ ...shown, ideas: [idea, ...shown.ideas] }));
      },
    });
  }

  function meToo(idea: api.Idea) {
    return sayingMeToo.run(() => api.meToo(login.accessToken, projectId, idea.id), {
      what: "say me too",
      done: (said) => {
        garden.change((shown) => ({ ...shown, ideas: shown.ideas.map((listed) => (listed.id === said.id ? said : listed)) }));
      },
    });
  }
</script>

<Back href={p("/")} text="your projects" />

<Loaded
  resource={garden}
  title="Garden"
  notFound={{ heading: "This project isn't here.", text: "It doesn't exist, or you aren't a member of it." }}
  failed="This garden can't be shown right now."
>
  {#snippet children(shown)}
    <Page title={shown.project.name} heading={shown.project.name}>
      <p class="quiet small"><a href={p("/projects/:id/settings", { params: { id: projectId } })}>settings</a></p>

      <form onsubmit={plant}>
        <!-- maxlength counts an emoji twice, so it stops a little before
             frogly's limit, but never past it -->
        <TextField
          id="idea-text"
          label="A new idea"
          bind:value={text}
          placeholder="plant an idea…"
          maxlength={api.maxIdeaLength}
          busy={planting.busy}
          problem={planting.problem}
          describedby="idea-count idea-problem"
        />
        <p class="count" id="idea-count">{text.length} / {api.maxIdeaLength}</p>
        <Problem id="idea-problem" text={planting.problem} />
      </form>

      <Section label="the garden" id="garden-label">
        {#if shown.ideas.length === 0}
          <p class="quiet">Nothing planted yet. Write down the first idea above.</p>
        {:else}
          <Rows>
            {#each shown.ideas as idea (idea.id)}
              <Row serif meta={`${idea.plantedBy} · ${plantedOn(idea.plantedAt)} · ${peopleBehind(idea.people)}`}>
                {idea.text}
                {#snippet actions()}
                  <Button busy={sayingMeToo.busy} onclick={() => meToo(idea)}>me too <span class="hidden">for {idea.text}</span></Button>
                {/snippet}
              </Row>
            {/each}
          </Rows>
        {/if}
        <Problem text={sayingMeToo.problem} />
      </Section>
    </Page>
  {/snippet}
</Loaded>

<style>
  .count {
    font-size: 13px;
    color: var(--ink-3);
    text-align: right;
    margin: 4px 0 0;
  }
</style>
