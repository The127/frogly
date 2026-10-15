<script lang="ts">
  import { UserManager } from "oidc-client-ts";
  import { onMount } from "svelte";

  import { fetchConfig, fetchMe } from "./api";
  import { loginSettings } from "./auth";
  import { type Session, resume } from "./session";

  let session = $state<Session | "loading" | "failed">("loading");
  let login: UserManager | undefined;

  onMount(async () => {
    try {
      login = new UserManager(loginSettings(await fetchConfig(), window.location.origin));
      session = await resume(login, window.location.pathname, fetchMe);

      // automaticSilentRenew failed, maybe because the refresh token expired
      login.events.addSilentRenewError(() => {
        session = { kind: "loggedOut" };
      });
    } catch (error) {
      console.error(error);
      session = "failed";
    } finally {
      // the callback URL holds a one-time code, a reload mustn't use it
      // again, whether the login worked or not
      if (window.location.pathname === "/callback") {
        window.history.replaceState(null, "", "/");
      }
    }
  });
</script>

<header class="top">
  <span class="mark"><span>蛙</span>frogly</span>
  {#if typeof session === "object" && session.kind === "loggedIn"}
    <button class="btn" onclick={() => login?.signoutRedirect()}>log out</button>
  {/if}
</header>

<main class="narrow">
  {#if session === "loading"}
    <p class="quiet">…</p>
  {:else if session === "failed"}
    <h1>frogly can't be reached right now.</h1>
    <p class="quiet">Try again in a moment.</p>
  {:else if session.kind === "loggedIn"}
    <h1>Hello, {session.name}</h1>
  {:else}
    <h1>A quiet place for your team's work.</h1>
    {#if session.loginFailed}
      <p class="quiet">The login didn't work. Try again.</p>
    {/if}
    <button class="btn pond" onclick={() => login?.signinRedirect()}>log in</button>
  {/if}
</main>
