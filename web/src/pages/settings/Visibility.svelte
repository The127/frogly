<script lang="ts">
  import { Action, type Busy } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { getLogin } from "../../lib/login";
  import Button from "../../ui/Button.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Section from "../../ui/Section.svelte";

  let {
    projectId,
    project,
    busy,
    changed,
    gone,
  }: { projectId: string; project: api.Project; busy: Busy; changed: (project: api.Project) => void; gone: () => void } = $props();

  const login = getLogin();
  // svelte-ignore state_referenced_locally (the page hands in one Busy for its whole life)
  const switching = new Action(login, () => gone(), busy);
  const isPublic = $derived(project.visibility === "public");

  function toggle() {
    return switching.run(() => api.changeVisibility(login.accessToken, projectId, isPublic ? "private" : "public"), {
      what: "change who sees the project",
      done: changed,
    });
  }
</script>

<Section label="who sees it" id="visibility-label">
  <p class="state">
    <!-- polite, so a screen reader says the new state once the change is saved -->
    <span class="quiet" aria-live="polite">
      {isPublic ? "Public, everyone in frogly sees its name." : "Private, only its members see it."}
    </span>
    <Button busy={switching.busy} onclick={toggle}>make it {isPublic ? "private" : "public"}</Button>
  </p>
  <Problem text={switching.problem} />
</Section>

<style>
  .state {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 14px;
    margin: 0;
  }
</style>
