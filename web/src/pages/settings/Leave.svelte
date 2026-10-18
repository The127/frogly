<script lang="ts">
  import { tick } from "svelte";

  import { Action, Busy } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { getLogin } from "../../lib/login";
  import { navigate } from "../../router";
  import Button from "../../ui/Button.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Section from "../../ui/Section.svelte";

  let { projectId, project, gone }: { projectId: string; project: api.ProjectWithMembers; gone: () => void } = $props();

  const login = getLogin();
  // checking what leaving does and leaving run one at a time
  const busy = new Busy();
  const checking = new Action(login, () => gone(), busy);
  const leaving = new Action(login, () => gone(), busy);

  // leaving takes two clicks, since you can't get back into a private or
  // closed project on your own
  let confirming = $state(false);
  let leaveButton = $state<HTMLButtonElement>();
  let stayButton = $state<HTMLButtonElement>();

  // loaded again when someone asks to leave, apart from the project the
  // page shows, so an unsaved pick of who can join stays
  let checked = $state<{ members: number; joining: api.Joining }>();
  const known = $derived(checked ?? { members: project.members, joining: project.joining });

  // a private project is always closed
  const comingBack: Record<api.Joining, string> = {
    open: "You can join it again whenever you like.",
    request: "You can come back once a member says yes when you ask, or adds you.",
    closed: "You can only come back if a member adds you.",
  };
  const note = $derived(
    known.members === 1
      ? "You're its last member. It turns public and open, so anyone in frogly can find it and join."
      : comingBack[known.joining],
  );

  async function confirm() {
    confirming = true;
    await tick();
    stayButton?.focus();
  }

  // not a member any more, like after leaving in another tab, is what
  // leaving was for
  async function home() {
    await navigate("/");
  }

  function askToLeave() {
    // others may have joined, left or changed who can join since the page
    // loaded, and the note says what leaving does now
    return checking.run(() => api.project(login.accessToken, projectId), {
      what: "check what leaving does",
      done: async (now) => {
        checked = { members: now.members, joining: now.joining };
        await confirm();
      },
      // what the page knew from before still makes a fair note
      failed: confirm,
      conflict: confirm,
      notFound: home,
    });
  }

  async function stay() {
    if (busy.now) {
      return;
    }

    leaving.problem = "";
    confirming = false;
    await tick();
    leaveButton?.focus();
  }

  function leave() {
    // still busy until the home page shows, so a second click can't send
    // another leave
    return leaving.run(() => api.leave(login.accessToken, projectId), {
      what: "take you out of this project",
      done: home,
      notFound: home,
    });
  }
</script>

<Section label="leave" id="leave-label">
  {#if confirming}
    <p class="quiet" id="coming-back">{note}</p>
  {/if}
  <div class="acts">
    {#if confirming}
      <Button busy={busy.now} onclick={leave} aria-describedby="coming-back">yes, leave {project.name}</Button>
      <Button pond busy={busy.now} onclick={stay} aria-describedby="coming-back" bind:ref={stayButton}>stay</Button>
    {:else}
      <Button busy={busy.now} onclick={askToLeave} bind:ref={leaveButton}>leave this project</Button>
    {/if}
  </div>
  <Problem text={leaving.problem || checking.problem} />
</Section>

<style>
  .acts {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-top: 8px;
  }
</style>
