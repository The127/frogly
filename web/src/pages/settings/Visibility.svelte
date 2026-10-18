<script lang="ts">
  import { Action, type Busy } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { visibilityText } from "../../lib/format";
  import { getLogin } from "../../lib/login";
  import Button from "../../ui/Button.svelte";
  import Notice from "../../ui/Notice.svelte";
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
  const switching = new Action(login, () => gone(), busy);
  const isPublic = $derived(project.visibility === "public");

  function toggle() {
    return switching.run(() => api.changeVisibility(login.accessToken, project.id, isPublic ? "private" : "public"), {
      what: "change who sees the project",
      done: changed,
    });
  }
</script>

<Section label="who sees it">
  <div class="state">
    <!-- a polite notice, so a screen reader says the new state once the
         change is saved -->
    <Notice text={visibilityText(project.visibility)} />
    <Button busy={switching.busy} onclick={toggle}>make it {isPublic ? "private" : "public"}</Button>
  </div>
  <Problem text={switching.problem} />
</Section>

<style>
  .state {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 14px;
  }

  .state :global(p) {
    margin: 0;
  }
</style>
