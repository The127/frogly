<script lang="ts">
  import { onMount } from "svelte";

  import { type Idea, type Project, type SessionEnded, maxIdeaLength } from "./api";
  import { addIdea, loadGarden, meToo, peopleBehind, plantedOn } from "./garden";
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
  let sayingMeToo = $state(false);
  let meTooProblem = $state("");

  onMount(async () => {
    const loaded = await loadGarden(login.accessToken, projectId);
    if (loaded.kind === "loggedOut") {
      login.onLoggedOut();
    } else if (loaded.kind === "notAllowed") {
      login.onNotAllowed();
    } else {
      garden = loaded;
    }
  });

  // what every action does when the project is gone or the session ended,
  // so each one only handles its own outcomes
  function handledElsewhere(outcome: { kind: "notFound" } | SessionEnded) {
    switch (outcome.kind) {
      case "notFound":
        garden = { kind: "notFound" };
        break;
      case "loggedOut":
        login.onLoggedOut();
        break;
      case "notAllowed":
        login.onNotAllowed();
    }
  }

  async function sayMeTooTo(idea: Idea) {
    if (sayingMeToo || garden.kind !== "listed") {
      return;
    }

    sayingMeToo = true;
    meTooProblem = "";

    const outcome = await meToo(login.accessToken, projectId, idea.id);
    sayingMeToo = false;

    switch (outcome.kind) {
      case "said":
        if (garden.kind === "listed") {
          garden = { ...garden, ideas: garden.ideas.map((listed) => (listed.id === idea.id ? outcome.idea : listed)) };
        }
        break;
      case "notSaid":
        meTooProblem = "frogly couldn't say me too. Try again in a moment.";
        break;
      default:
        handledElsewhere(outcome);
    }
  }

  async function plant(event: SubmitEvent) {
    event.preventDefault();
    if (saving || garden.kind !== "listed") {
      return;
    }

    saving = true;
    problem = "";

    const outcome = await addIdea(login.accessToken, projectId, text);
    saving = false;

    switch (outcome.kind) {
      case "saved":
        text = "";
        garden = { ...garden, ideas: [outcome.idea, ...garden.ideas] };
        break;
      case "refused":
        problem = outcome.problem;
        break;
      case "notSaved":
        problem = "frogly couldn't plant the idea. Try again in a moment.";
        break;
      default:
        handledElsewhere(outcome);
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
  <p class="quiet"><a href={p("/projects/:id/settings", { params: { id: projectId } })}>settings</a></p>

  <form onsubmit={plant}>
    <label class="hidden" for="idea-text">A new idea</label>
    <!-- readonly while saving, since a disabled field would lose the focus.
         maxlength counts an emoji twice, so it stops a little before
         frogly's limit, but never past it. -->
    <input
      class="plant"
      id="idea-text"
      bind:value={text}
      placeholder="plant an idea…"
      maxlength={maxIdeaLength}
      readonly={saving}
      aria-busy={saving}
      aria-invalid={problem ? "true" : undefined}
      aria-describedby="idea-count idea-problem"
    />
    <p class="count" id="idea-count">{text.length} / {maxIdeaLength}</p>
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
            <span>
              {idea.text}
              <span class="meta">{idea.plantedBy} · {plantedOn(idea.plantedAt)} · {peopleBehind(idea.people)}</span>
            </span>
            <button class="btn" onclick={() => sayMeTooTo(idea)} aria-disabled={sayingMeToo}>
              me too <span class="hidden">for {idea.text}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{meTooProblem}</p>
  </section>
{/if}
