<script lang="ts">
  import { onMount } from "svelte";

  import type { Person, Project, SessionEnded } from "./api";
  import { getLogin } from "./login";
  import { p } from "./router";
  import { addPerson, findPeople, loadSettings, switchVisibility } from "./settings";

  let { projectId }: { projectId: string } = $props();

  const login = getLogin();

  let settings = $state<{ kind: "loading" } | { kind: "loaded"; project: Project } | { kind: "notFound" } | { kind: "failed" }>({
    kind: "loading",
  });
  const isPublic = $derived(settings.kind === "loaded" && settings.project.visibility === "public");
  let switching = $state(false);
  let problem = $state("");

  let searchText = $state("");
  let people = $state<Person[] | undefined>();
  // one flag for searching and adding, so a search that finishes late can't
  // bring back someone just added
  let busy = $state(false);
  let peopleProblem = $state("");
  let addProblem = $state("");
  let added = $state("");
  let searchField = $state<HTMLInputElement>();
  const foundText = $derived(
    people === undefined ? "" : `${people.length} ${people.length === 1 ? "person" : "people"} found`,
  );

  onMount(async () => {
    const loaded = await loadSettings(login.accessToken, projectId);
    if (loaded.kind === "loggedOut") {
      login.onLoggedOut();
    } else if (loaded.kind === "notAllowed") {
      login.onNotAllowed();
    } else {
      settings = loaded;
    }
  });

  // what every action does when the project is gone or the session ended,
  // so each one only handles its own outcomes
  function handledElsewhere(outcome: { kind: "notFound" } | SessionEnded) {
    switch (outcome.kind) {
      case "notFound":
        settings = { kind: "notFound" };
        break;
      case "loggedOut":
        login.onLoggedOut();
        break;
      case "notAllowed":
        login.onNotAllowed();
    }
  }

  async function toggleVisibility() {
    if (switching || settings.kind !== "loaded") {
      return;
    }

    switching = true;
    problem = "";

    const outcome = await switchVisibility(login.accessToken, projectId, isPublic ? "private" : "public");
    switching = false;

    switch (outcome.kind) {
      case "changed":
        settings = { kind: "loaded", project: outcome.project };
        break;
      case "notChanged":
        problem = "frogly couldn't change who sees the project. Try again in a moment.";
        break;
      default:
        handledElsewhere(outcome);
    }
  }

  async function search(event: SubmitEvent) {
    event.preventDefault();
    if (busy) {
      return;
    }

    busy = true;
    peopleProblem = "";
    addProblem = "";
    added = "";

    const outcome = await findPeople(login.accessToken, projectId, searchText);
    busy = false;

    switch (outcome.kind) {
      case "found":
        people = outcome.people;
        break;
      case "refused":
        peopleProblem = outcome.problem;
        break;
      case "failed":
        peopleProblem = "frogly couldn't search right now. Try again in a moment.";
        break;
      default:
        handledElsewhere(outcome);
    }
  }

  async function add(person: Person) {
    if (busy) {
      return;
    }

    busy = true;
    addProblem = "";
    added = "";

    const outcome = await addPerson(login.accessToken, projectId, person.id);
    busy = false;

    switch (outcome.kind) {
      case "added":
        people = people?.filter((found) => found.id !== person.id);
        added = `${outcome.person.name} is a member now.`;
        // the button that had the focus is gone with the person
        searchField?.focus();
        break;
      case "notAdded":
        addProblem = `frogly couldn't add ${person.name}. Try again in a moment.`;
        break;
      default:
        handledElsewhere(outcome);
    }
  }
</script>

<p class="quiet">
  <a href={p("/projects/:id", { params: { id: projectId } })}><span aria-hidden="true">←</span> the garden</a>
</p>

{#if settings.kind === "loading"}
  <h1 class="quiet">…</h1>
{:else if settings.kind === "notFound"}
  <h1>This project isn't here.</h1>
  <p class="quiet">It doesn't exist, or you aren't a member of it.</p>
{:else if settings.kind === "failed"}
  <h1>These settings can't be shown right now.</h1>
  <p class="quiet">Reload the page in a moment.</p>
{:else}
  <h1>Settings for {settings.project.name}</h1>

  <section aria-labelledby="visibility-label">
    <p class="label" id="visibility-label">who sees it</p>
    <p class="quiet visibility">
      <!-- polite, so a screen reader says the new state once the change is saved -->
      <span aria-live="polite">
        {isPublic ? "Public, everyone in frogly sees its name." : "Private, only its members see it."}
      </span>
      <button class="btn" onclick={toggleVisibility}>
        make it {isPublic ? "private" : "public"}
      </button>
    </p>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{problem}</p>
  </section>

  <section aria-labelledby="people-label">
    <p class="label" id="people-label">add someone</p>
    <form onsubmit={search}>
      <label class="hidden" for="people-search">Name or email of someone who uses frogly</label>
      <!-- readonly while searching, since a disabled field would lose the focus -->
      <input
        class="plant"
        id="people-search"
        bind:this={searchField}
        bind:value={searchText}
        placeholder="find someone by name or email…"
        readonly={busy}
        aria-busy={busy}
        aria-invalid={peopleProblem ? "true" : undefined}
        aria-describedby="people-problem"
      />
    </form>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" id="people-problem" role="alert">{peopleProblem}</p>

    <!-- polite, so a screen reader hears what the search found -->
    <p class="hidden" aria-live="polite">
      {foundText}
    </p>
    {#if people !== undefined}
      {#if people.length === 0}
        <p class="quiet">Nobody found who could join.</p>
      {:else}
        <ul class="people">
          {#each people as person (person.id)}
            <li>
              <span>{person.name} <span class="meta">{person.email}</span></span>
              <button class="btn" onclick={() => add(person)}>add <span class="hidden">{person.name}</span></button>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{addProblem}</p>
    <p class="quiet" aria-live="polite">{added}</p>
  </section>
{/if}
