<script lang="ts">
  import { onMount } from "svelte";

  import { type Idea, type Project, maxIdeaLength } from "./api";
  import { addIdea, ideaLength, loadGarden, plantedOn } from "./garden";
  import { getLogin } from "./login";
  import { p } from "./router";

  let { projectId }: { projectId: string } = $props();

  const login = getLogin();

  let garden = $state<
    { kind: "loading" } | { kind: "listed"; project: Project; ideas: Idea[] } | { kind: "notFound" } | { kind: "failed" }
  >({ kind: "loading" });
  let text = $state("");
  let problem = $state("");
  let saving = $state(false);
  const length = $derived(ideaLength(text));
  const tooLong = $derived(length > maxIdeaLength);

  onMount(async () => {
    const loaded = await loadGarden(login.accessToken, projectId);
    if (loaded.kind === "loggedOut") {
      login.onLoggedOut();
    } else {
      garden = loaded;
    }
  });

  async function plant(event: SubmitEvent) {
    event.preventDefault();
    if (saving || garden.kind !== "listed") {
      return;
    }

    // the same reason frogly gives, without asking it
    if (tooLong) {
      problem = `an idea has at most ${maxIdeaLength} characters`;

      return;
    }

    saving = true;
    problem = "";

    const added = await addIdea(login.accessToken, projectId, text);
    saving = false;

    switch (added.kind) {
      case "saved":
        text = "";
        garden = { ...garden, ideas: [added.idea, ...garden.ideas] };
        break;
      case "refused":
        problem = added.problem;
        break;
      case "notSaved":
        problem = "frogly couldn't plant the idea. Try again in a moment.";
        break;
      case "notFound":
        garden = { kind: "notFound" };
        break;
      case "loggedOut":
        login.onLoggedOut();
    }
  }
</script>

<p class="quiet"><a href={p("/")}><span aria-hidden="true">←</span> your projects</a></p>

{#if garden.kind === "loading"}
  <h1 class="quiet">…</h1>
{:else if garden.kind === "notFound"}
  <h1>This project isn't here.</h1>
  <p class="quiet">It doesn't exist, or you aren't a member of it.</p>
{:else if garden.kind === "failed"}
  <h1>This garden can't be shown right now.</h1>
  <p class="quiet">Reload the page in a moment.</p>
{:else}
  <h1>{garden.project.name}</h1>

  <form onsubmit={plant}>
    <label class="hidden" for="idea-text">A new idea</label>
    <!-- readonly while saving, since a disabled field would lose the focus -->
    <input
      class="plant"
      id="idea-text"
      bind:value={text}
      placeholder="plant an idea…"
      readonly={saving}
      aria-busy={saving}
      aria-invalid={problem || tooLong ? "true" : undefined}
      aria-describedby="idea-count idea-problem"
    />
    <p class="count" class:over={tooLong} id="idea-count">{length} / {maxIdeaLength}</p>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" id="idea-problem" role="alert">{problem}</p>
  </form>

  <section aria-labelledby="garden-label">
    <p class="label" id="garden-label">the garden</p>
    {#if garden.ideas.length === 0}
      <p class="quiet">Nothing planted yet. Write down the first idea above.</p>
    {:else}
      <ul class="ideas">
        {#each garden.ideas as idea (idea.id)}
          <li>
            {idea.text}
            <span class="meta">{idea.plantedBy} · {plantedOn(idea.plantedAt)}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}
