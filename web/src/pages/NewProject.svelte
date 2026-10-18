<script lang="ts">
  import { Action } from "../lib/action.svelte";
  import * as api from "../lib/api";
  import { joiningChoices, nameFrom, visibilityChoices } from "../lib/format";
  import { getLogin } from "../lib/login";
  import { navigate, p } from "../router";
  import Back from "../ui/Back.svelte";
  import Button from "../ui/Button.svelte";
  import Choice from "../ui/Choice.svelte";
  import Page from "../ui/Page.svelte";
  import Problem from "../ui/Problem.svelte";
  import TextField from "../ui/TextField.svelte";

  const login = getLogin();
  // a new project belongs to nothing that could be gone, so a 404 is
  // frogly's problem like any other failure
  const create = new Action(login);

  let name = $state(nameFrom(window.location.search));
  // nothing is chosen in advance, the person decides who sees a project
  let visibility = $state<api.Visibility | undefined>();
  let joining = $state<api.Joining | undefined>();

  function submit(event: SubmitEvent) {
    event.preventDefault();
    // the form keeps a join choice after a switch to private, which frogly
    // would refuse
    const join = visibility === "public" ? joining : undefined;

    return create.run(() => api.createProject(login.accessToken, name, visibility, join), {
      what: "save the project",
      done: async (project) => {
        await navigate("/projects/:id", { params: { id: project.id } });
      },
    });
  }
</script>

<Back href={p("/")} text="your projects" />

<Page title="New project" heading="New project">
  <form onsubmit={submit}>
    <div class="field">
      <label class="label" for="project-name">name</label>
      <TextField id="project-name" bind:value={name} busy={create.busy} problem={create.problem} describedby="project-problem" />
    </div>

    <div class="field">
      <Choice legend="who sees it" name="visibility" choices={visibilityChoices} bind:group={visibility} describedby="project-problem" />
    </div>

    {#if visibility === "public"}
      <div class="field">
        <Choice legend="who can join" name="joining" choices={joiningChoices} bind:group={joining} describedby="project-problem" />
      </div>
    {/if}

    <div class="field">
      <Button pond type="submit" busy={create.busy}>create</Button>
      <Problem id="project-problem" text={create.problem} />
    </div>
  </form>
</Page>

<style>
  .field {
    margin-top: 32px;
  }

  label {
    display: block;
    margin-bottom: 4px;
  }
</style>
