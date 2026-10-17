<script lang="ts">
  import { onMount } from "svelte";

  import type { Project } from "./api";
  import { getLogin } from "./login";
  import { p } from "./router";
  import { loadSettings, switchVisibility } from "./settings";

  let { projectId }: { projectId: string } = $props();

  const login = getLogin();

  let settings = $state<{ kind: "loading" } | { kind: "loaded"; project: Project } | { kind: "notFound" } | { kind: "failed" }>({
    kind: "loading",
  });
  let switching = $state(false);
  let problem = $state("");

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

    const other = settings.project.visibility === "public" ? "private" : "public";
    const switched = await switchVisibility(login.accessToken, projectId, other);
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
  <h1>{settings.project.name}</h1>

  <section aria-labelledby="visibility-label">
    <p class="label" id="visibility-label">who sees it</p>
    <p class="quiet visibility">
      <!-- polite, so a screen reader says the new state once the change is saved -->
      <span aria-live="polite">
        {settings.project.visibility === "public" ? "Public, everyone in frogly sees its name." : "Private, only its members see it."}
      </span>
      <button class="btn" onclick={toggleVisibility}>
        make it {settings.project.visibility === "public" ? "private" : "public"}
      </button>
    </p>
    <!-- always there, so screen readers announce the text when it changes -->
    <p class="problem" role="alert">{problem}</p>
  </section>
{/if}
