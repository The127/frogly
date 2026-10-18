<script lang="ts">
  import { onMount, tick } from "svelte";

  import type { Joining, Person, Project, SessionEnded, WaitingRequest } from "./api";
  import { getLogin } from "./login";
  import { navigate, p } from "./router";
  import { addPerson, answerRequest, findPeople, leave, loadRequests, loadSettings, switchJoining, switchVisibility } from "./settings";

  let { projectId }: { projectId: string } = $props();

  const login = getLogin();

  let settings = $state<{ kind: "loading" } | { kind: "loaded"; project: Project } | { kind: "notFound" } | { kind: "failed" }>({
    kind: "loading",
  });
  const isPublic = $derived(settings.kind === "loaded" && settings.project.visibility === "public");
  let switching = $state(false);
  let problem = $state("");
  let joiningProblem = $state("");
  // picking a choice and saving it are two steps, since arrow keys walk
  // through every choice on the way, and saving each would let people in
  // for a moment
  let chosenJoining = $state<Joining | undefined>();
  const joiningChoices: { value: Joining; text: string }[] = [
    { value: "open", text: "anyone, right away" },
    { value: "request", text: "people who ask, once a member says yes" },
    { value: "closed", text: "only people a member adds" },
  ];

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

  let requests = $state<WaitingRequest[]>([]);
  let answering = $state(false);
  let requestsProblem = $state("");
  let answered = $state("");
  let requestsHeading = $state<HTMLElement>();

  // leaving takes two clicks, since you can't get back into a private or
  // closed project on your own
  let confirmingLeave = $state(false);
  let leaving = $state(false);
  let leaveProblem = $state("");
  let leaveButton = $state<HTMLButtonElement>();
  let stayButton = $state<HTMLButtonElement>();
  // counted again when someone asks to leave
  let members = $state(0);
  // a private project is always closed
  const comingBack: Record<Joining, string> = {
    open: "You can join it again whenever you like.",
    request: "You can come back once a member says yes when you ask, or adds you.",
    closed: "You can only come back if a member adds you.",
  };

  onMount(async () => {
    // both at once, so the requests don't push the page down later
    const [loaded, waiting] = await Promise.all([
      loadSettings(login.accessToken, projectId),
      loadRequests(login.accessToken, projectId),
    ]);
    if (loaded.kind === "loggedOut") {
      login.onLoggedOut();
    } else if (loaded.kind === "notAllowed") {
      login.onNotAllowed();
    } else if (loaded.kind === "loaded") {
      show(loaded.project);
      members = loaded.members;
      showRequests(waiting);
    } else {
      settings = loaded;
    }
  });

  async function refreshRequests() {
    showRequests(await loadRequests(login.accessToken, projectId));
  }

  function showRequests(loaded: Awaited<ReturnType<typeof loadRequests>>) {
    switch (loaded.kind) {
      case "listed":
        requests = loaded.requests;
        break;
      case "failed":
        requestsProblem = "frogly can't show who asked to join right now. Reload the page in a moment.";
        break;
      default:
        handledElsewhere(loaded);
    }
  }

  async function answer(request: WaitingRequest, reply: "yes" | "no") {
    if (answering) {
      return;
    }

    answering = true;
    requestsProblem = "";
    answered = "";

    const outcome = await answerRequest(login.accessToken, projectId, request.id, reply);
    answering = false;

    switch (outcome.kind) {
      case "answered":
        requests = requests.filter((waiting) => waiting.id !== request.id);
        answered = reply === "yes" ? `${request.name} is a member now.` : `You said no to ${request.name}.`;
        // the button that had the focus is gone with the request
        requestsHeading?.focus();
        break;
      case "notAnswered":
        requestsProblem = `frogly couldn't answer ${request.name}. Try again in a moment.`;
        break;
      case "notFound":
        requestsProblem = `${request.name}'s request was answered already.`;
        // another member answered it, and the list shows what's left
        await refreshRequests();
        break;
      default:
        handledElsewhere(outcome);
    }
  }

  // a fresh project from frogly also starts the pick from what's saved
  function show(project: Project) {
    settings = { kind: "loaded", project };
    chosenJoining = project.joining;
    // frogly drops the requests of a project that doesn't take them any more
    if (project.joining !== "request") {
      requests = [];
    }
  }

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
    joiningProblem = "";

    const outcome = await switchVisibility(login.accessToken, projectId, isPublic ? "private" : "public");
    switching = false;

    switch (outcome.kind) {
      case "changed":
        show(outcome.project);
        break;
      case "notChanged":
        problem = "frogly couldn't change who sees the project. Try again in a moment.";
        break;
      default:
        handledElsewhere(outcome);
    }
  }

  async function saveJoining() {
    if (switching || settings.kind !== "loaded" || chosenJoining === undefined) {
      return;
    }

    switching = true;
    problem = "";
    joiningProblem = "";

    const outcome = await switchJoining(login.accessToken, projectId, chosenJoining);
    switching = false;

    switch (outcome.kind) {
      case "changed":
        show(outcome.project);
        break;
      case "notChanged":
        joiningProblem = "frogly couldn't change who can join the project. Try again in a moment.";
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
    // empties the count, so the same number found again is announced again
    people = undefined;

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

  async function askToLeave() {
    if (leaving) {
      return;
    }

    // others may have joined or left since the page loaded, and the note
    // tells the last member that the project opens
    leaving = true;
    const loaded = await loadSettings(login.accessToken, projectId);
    leaving = false;

    switch (loaded.kind) {
      case "loaded":
        members = loaded.members;
        break;
      // the count from before still makes a fair note
      case "failed":
        break;
      default:
        handledElsewhere(loaded);
        return;
    }

    confirmingLeave = true;
    await tick();
    stayButton?.focus();
  }

  async function stay() {
    if (leaving) {
      return;
    }

    leaveProblem = "";
    confirmingLeave = false;
    await tick();
    leaveButton?.focus();
  }

  async function leaveForGood() {
    if (leaving) {
      return;
    }

    leaving = true;
    leaveProblem = "";

    const outcome = await leave(login.accessToken, projectId);

    switch (outcome.kind) {
      // still leaving until the home page shows, so a second click can't
      // send another leave
      case "left":
        void navigate("/");
        break;
      case "notLeft":
        leaving = false;
        leaveProblem = "frogly couldn't take you out of this project. Try again in a moment.";
        break;
      default:
        leaving = false;
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
      <button class="btn" onclick={toggleVisibility} aria-disabled={switching}>
        make it {isPublic ? "private" : "public"}
      </button>
    </p>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{problem}</p>
  </section>

  {#if isPublic}
    <section>
      <fieldset class="choice" aria-describedby="joining-problem">
        <legend class="label">who can join</legend>
        {#each joiningChoices as choice (choice.value)}
          <label><input type="radio" name="joining" value={choice.value} bind:group={chosenJoining} /> {choice.text}</label>
        {/each}
      </fieldset>
      {#if chosenJoining !== settings.project.joining}
        <div class="acts">
          <button class="btn pond" onclick={saveJoining} aria-disabled={switching}>save who can join</button>
        </div>
      {/if}
      <!-- always there, so screen readers announce the text when it changes -->
      <p class="problem" id="joining-problem" role="alert">{joiningProblem}</p>
    </section>

    <section aria-labelledby="requests-label">
      <p class="label" id="requests-label" tabindex="-1" bind:this={requestsHeading}>asked to join</p>
      {#if requests.length === 0}
        <p class="quiet">Nobody waits for an answer right now.</p>
      {/if}
      <ul class="rows">
        {#each requests as request (request.id)}
          <li>
            <span>{request.name} <span class="meta">{request.email}</span></span>
            <span>
              <button class="btn pond" onclick={() => answer(request, "yes")} aria-disabled={answering}>
                yes <span class="hidden">to {request.name}</span>
              </button>
              <button class="btn" onclick={() => answer(request, "no")} aria-disabled={answering}>
                no <span class="hidden">to {request.name}</span>
              </button>
            </span>
          </li>
        {/each}
      </ul>
      <!-- always there, so screen readers announce the text when it changes -->
      <p class="problem" role="alert">{requestsProblem}</p>
      <p class="quiet" aria-live="polite">{answered}</p>
    </section>
  {/if}

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
        <ul class="rows">
          {#each people as person (person.id)}
            <li>
              <span>{person.name} <span class="meta">{person.email}</span></span>
              <button class="btn" onclick={() => add(person)} aria-disabled={busy}>add <span class="hidden">{person.name}</span></button>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{addProblem}</p>
    <p class="quiet" aria-live="polite">{added}</p>
  </section>

  <section aria-labelledby="leave-label">
    <p class="label" id="leave-label">leave</p>
    {#if confirmingLeave}
      <p class="quiet" id="coming-back">
        {members === 1
          ? "You're its last member. It turns public and open, so anyone in frogly can find it and join."
          : comingBack[settings.project.joining]}
      </p>
    {/if}
    <div class="acts">
      {#if confirmingLeave}
        <button class="btn" onclick={leaveForGood} aria-disabled={leaving} aria-describedby="coming-back">yes, leave {settings.project.name}</button>
        <button class="btn pond" bind:this={stayButton} onclick={stay} aria-disabled={leaving} aria-describedby="coming-back">stay</button>
      {:else}
        <button class="btn" bind:this={leaveButton} onclick={askToLeave} aria-disabled={leaving}>leave this project</button>
      {/if}
    </div>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{leaveProblem}</p>
  </section>
{/if}
