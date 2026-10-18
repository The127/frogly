<script lang="ts">
  import { Action, type Busy } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { joiningChoices } from "../../lib/format";
  import { getLogin } from "../../lib/login";
  import Button from "../../ui/Button.svelte";
  import Choice from "../../ui/Choice.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Section from "../../ui/Section.svelte";

  let {
    project,
    busy,
    changed,
    gone,
  }: { project: api.Project; busy: Busy; changed: (project: api.Project) => void; gone: () => void } = $props();

  const login = getLogin();
  // svelte-ignore state_referenced_locally (the page hands in one Busy for its whole life)
  const saving = new Action(login, () => gone(), busy);

  // picking a choice and saving it are two steps, since arrow keys walk
  // through every choice on the way, and saving each would let people in
  // for a moment. A fresh project from frogly starts the pick again from
  // what's saved.
  let chosen = $derived<api.Joining>(project.joining);

  function save() {
    const joining = chosen;

    return saving.run(() => api.changeJoining(login.accessToken, project.id, joining), {
      what: "change who can join the project",
      done: changed,
    });
  }
</script>

<!-- the choice's legend is the heading -->
<Section>
  <Choice legend="who can join" name="joining" choices={joiningChoices} bind:group={chosen} describedby="joining-problem" />
  {#if chosen !== project.joining}
    <div class="save">
      <Button pond busy={saving.busy} onclick={save}>save who can join</Button>
    </div>
  {/if}
  <Problem id="joining-problem" text={saving.problem} />
</Section>

<style>
  .save {
    margin-top: 20px;
  }
</style>
