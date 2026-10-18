<script lang="ts">
  import type { Joining, Visibility } from "./api";
  import { getLogin } from "./login";
  import { addProject, nameFrom } from "./projects";
  import { navigate, p } from "./router";

  const login = getLogin();

  let name = $state(nameFrom(window.location.search));
  // nothing is chosen in advance, the person decides who sees a project
  let visibility = $state<Visibility | undefined>();
  let joining = $state<Joining | undefined>();
  let problem = $state("");
  let saving = $state(false);

  async function create(event: SubmitEvent) {
    event.preventDefault();
    if (saving) {
      return;
    }

    saving = true;
    problem = "";

    const added = await addProject(login.accessToken, name, visibility, joining);
    saving = false;

    switch (added.kind) {
      case "saved":
        void navigate("/projects/:id", { params: { id: added.project.id } });
        break;
      case "refused":
        problem = added.problem;
        break;
      case "notSaved":
        problem = "frogly couldn't save the project. Try again in a moment.";
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

<h1>New project</h1>

<form class="fields" onsubmit={create}>
  <label class="label" for="project-name">name</label>
  <!-- readonly while saving, since a disabled field would lose the focus -->
  <input
    class="plant"
    id="project-name"
    bind:value={name}
    readonly={saving}
    aria-busy={saving}
    aria-invalid={problem ? "true" : undefined}
    aria-describedby="project-problem"
  />

  <!-- not disabled while saving, a disabled radio would lose the focus.
       create() already refuses a second save. -->
  <fieldset class="choice" aria-describedby="project-problem">
    <legend class="label">who sees it</legend>
    <label><input type="radio" name="visibility" value="private" bind:group={visibility} /> private, only its members see it</label>
    <label><input type="radio" name="visibility" value="public" bind:group={visibility} /> public, everyone in frogly sees its name</label>
  </fieldset>

  {#if visibility === "public"}
    <fieldset class="choice" aria-describedby="project-problem">
      <legend class="label">who can join</legend>
      <label><input type="radio" name="joining" value="open" bind:group={joining} /> anyone, right away</label>
      <label><input type="radio" name="joining" value="request" bind:group={joining} /> people who ask, once a member says yes</label>
      <label><input type="radio" name="joining" value="closed" bind:group={joining} /> only people a member adds</label>
    </fieldset>
  {/if}

  <div class="acts">
    <button class="btn pond" type="submit">create</button>
  </div>
  <!-- always there, so screen readers announce the text when it changes -->
  <p class="problem" id="project-problem" role="alert">{problem}</p>
</form>
