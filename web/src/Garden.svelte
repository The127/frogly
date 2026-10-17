<script lang="ts">
  import { onMount } from "svelte";

  import { type Idea, type Project, maxIdeaLength } from "./api";
  import { addIdea, loadGarden, plantedOn, switchVisibility } from "./garden";
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
  let switching = $state(false);
  let visibilityProblem = $state("");

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

  async function toggleVisibility() {
    if (switching || garden.kind !== "listed") {
      return;
    }

    switching = true;
    visibilityProblem = "";

    const other = garden.project.visibility === "public" ? "private" : "public";
    const switched = await switchVisibility(login.accessToken, projectId, other);
    switching = false;

    switch (switched.kind) {
      case "changed":
        if (garden.kind === "listed") {
          garden = { ...garden, project: switched.project };
        }
        break;
      case "notChanged":
        visibilityProblem = "frogly couldn't change who sees the project. Try again in a moment.";
        break;
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

  async function plant(event: SubmitEvent) {
    event.preventDefault();
    if (saving || garden.kind !== "listed") {
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
        break;
      case "notAllowed":
        login.onNotAllowed();
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
  <p class="quiet visibility">
    {garden.project.visibility === "public" ? "Public, everyone in frogly sees its name." : "Private, only its members see it."}
    <button class="btn" onclick={toggleVisibility} aria-busy={switching}>
      make it {garden.project.visibility === "public" ? "private" : "public"}
    </button>
  </p>
  <!-- always there, so screen readers announce the text when it changes -->
  <p class="problem" role="alert">{visibilityProblem}</p>

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
            {idea.text}
            <span class="meta">{idea.plantedBy} · {plantedOn(idea.plantedAt)}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}
