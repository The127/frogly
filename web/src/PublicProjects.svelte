<script lang="ts">
  import { onDestroy, onMount } from "svelte";

  import type { PublicProject, SessionEnded } from "./api";
  import { getLogin } from "./login";
  import { joinPublicProject, loadPublicProjects } from "./projects";
  import { navigate } from "./router";

  const { accessToken, onLoggedOut, onNotAllowed } = getLogin();

  let projects = $state<PublicProject[] | undefined>();
  let listFailed = $state(false);
  let joining = $state(false);
  let joinProblem = $state("");
  // a join can finish after the person moved on, and mustn't pull them back
  let gone = false;
  onDestroy(() => {
    gone = true;
  });

  function handledElsewhere(outcome: SessionEnded) {
    if (outcome.kind === "loggedOut") {
      onLoggedOut();
    } else {
      onNotAllowed();
    }
  }

  onMount(async () => {
    const loaded = await loadPublicProjects(accessToken);
    switch (loaded.kind) {
      case "listed":
        projects = loaded.projects;
        break;
      case "failed":
        listFailed = true;
        break;
      default:
        handledElsewhere(loaded);
    }
  });

  async function join(project: PublicProject) {
    if (joining) {
      return;
    }

    joining = true;
    joinProblem = "";

    const outcome = await joinPublicProject(accessToken, project.id);
    joining = false;

    switch (outcome.kind) {
      case "joined":
        if (!gone) {
          void navigate("/projects/:id", { params: { id: outcome.project.id } });
        }
        break;
      case "notJoined":
        joinProblem = `frogly couldn't join ${project.name}. Try again in a moment.`;
        break;
      case "notFound":
        projects = projects?.filter((listed) => listed.id !== project.id);
        joinProblem = `${project.name} can't be joined any more.`;
        break;
      default:
        handledElsewhere(outcome);
    }
  }
</script>

<section aria-labelledby="public-label">
  <p class="label" id="public-label">public projects of others</p>

  {#if listFailed}
    <p class="quiet">frogly can't list them right now. Reload the page in a moment.</p>
  {:else if projects === undefined}
    <p class="quiet">…</p>
  {:else if projects.length === 0}
    <p class="quiet">Nobody else has a public project yet.</p>
  {:else}
    <!-- not links, a garden opens to its members -->
    <ul class="rows">
      {#each projects as project (project.id)}
        <li>
          <span>{project.name}</span>
          {#if project.joining === "open"}
            <button class="btn" onclick={() => join(project)} aria-disabled={joining}>
              join <span class="hidden">{project.name}</span>
            </button>
          {:else}
            <span class="meta">a member adds people</span>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
  <!-- always there, so screen readers announce the text when it changes -->
  <p class="problem" role="alert">{joinProblem}</p>
</section>
