<script lang="ts">
  import type { Action } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { joiningChoices } from "../../lib/format";
  import { getLogin } from "../../lib/login";
  import Button from "../../ui/Button.svelte";
  import Choice from "../../ui/Choice.svelte";
  import Problem from "../../ui/Problem.svelte";

  let {
    projectId,
    project,
    action,
    other,
    changed,
  }: { projectId: string; project: api.Project; action: Action; other: Action; changed: (project: api.Project) => void } = $props();

  const login = getLogin();

  // picking a choice and saving it are two steps, since arrow keys walk
  // through every choice on the way, and saving each would let people in
  // for a moment
  let chosen = $state<api.Joining>();
  // a fresh project from frogly starts the pick from what's saved
  $effect.pre(() => {
    chosen = project.joining;
  });

  function save() {
    const joining = chosen;
    if (joining === undefined) {
      return;
    }

    other.problem = "";

    return action.run(() => api.changeJoining(login.accessToken, projectId, joining), {
      what: "change who can join the project",
      done: changed,
    });
  }
</script>

<section class="joining">
  <Choice legend="who can join" name="joining" choices={joiningChoices} bind:group={chosen} describedby="joining-problem" />
  {#if chosen !== project.joining}
    <div class="save">
      <Button pond busy={action.busy} onclick={save}>save who can join</Button>
    </div>
  {/if}
  <Problem id="joining-problem" text={action.problem} />
</section>

<style>
  .joining {
    margin-top: 44px;
  }

  .save {
    margin-top: 20px;
  }
</style>
