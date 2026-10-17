<script lang="ts">
  import { onMount } from "svelte";

  import type { Person, Project } from "./api";
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
  let searching = $state(false);
  let adding = $state(false);
  let peopleProblem = $state("");
  let added = $state("");

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

  async function toggleVisibility() {
    if (switching || settings.kind !== "loaded") {
      return;
    }

    switching = true;
    problem = "";

    const switched = await switchVisibility(login.accessToken, projectId, isPublic ? "private" : "public");
    switching = false;

    switch (switched.kind) {
      case "changed":
        settings = { kind: "loaded", project: switched.project };
        break;
      case "notChanged":
        problem = "frogly couldn't change who sees the project. Try again in a moment.";
        break;
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
  async function search(event: SubmitEvent) {
    event.preventDefault();
    if (searching) {
      return;
    }

    searching = true;
    peopleProblem = "";

    const found = await findPeople(login.accessToken, projectId, searchText);
    searching = false;

    switch (found.kind) {
      case "found":
        people = found.people;
        break;
      case "refused":
        peopleProblem = found.problem;
        break;
      case "failed":
        peopleProblem = "frogly couldn't search right now. Try again in a moment.";
        break;
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

  async function add(person: Person) {
    if (adding) {
      return;
    }

    adding = true;
    peopleProblem = "";

    const outcome = await addPerson(login.accessToken, projectId, person.id);
    adding = false;

    switch (outcome.kind) {
      case "added":
        people = people?.filter((found) => found.id !== person.id);
        added = `${outcome.person.name} is a member now.`;
        break;
      case "notAdded":
        peopleProblem = `frogly couldn't add ${person.name}. Try again in a moment.`;
        break;
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
        bind:value={searchText}
        placeholder="find someone by name or email…"
        readonly={searching}
        aria-busy={searching}
        aria-invalid={peopleProblem ? "true" : undefined}
        aria-describedby="people-problem"
      />
    </form>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" id="people-problem" role="alert">{peopleProblem}</p>

    {#if people !== undefined}
      {#if people.length === 0}
        <p class="quiet">Nobody found who isn't a member already.</p>
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
    <p class="quiet" aria-live="polite">{added}</p>
  </section>
{/if}
