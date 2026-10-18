<script lang="ts">
  import { onMount } from "svelte";

  import { Action, Resource } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { both, map } from "../../lib/client";
  import { peopleBehind, plantedOn, projectGone } from "../../lib/format";
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

    return map(loaded, ([project, ideas]) => ({ project, ideas }));
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
  notFound={projectGone}
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
          aria-describedby="idea-count"
        />
        <div class="under">
          <p class="quiet small" id="idea-count">{text.length} / {api.maxIdeaLength}</p>
          <Button pond type="submit" busy={planting.busy}>plant</Button>
        </div>
      </form>

      <Section label="the garden">
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
  .under {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }

  .under p {
    margin: 0;
  }
</style>
