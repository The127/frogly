<script lang="ts">
  import { UserManager } from "oidc-client-ts";
  import { onMount } from "svelte";

  import * as api from "./lib/api";
  import { callbackPath, loginSettings } from "./lib/auth";
  import { LoggedOut } from "./lib/client";
  import { setLogin } from "./lib/login";
  import { type Session, resume } from "./lib/session";
  import { Router, p } from "./router";
  import Button from "./ui/Button.svelte";
  import Loading from "./ui/Loading.svelte";
  import Page from "./ui/Page.svelte";

  let session = $state<Session | { kind: "loading" } | { kind: "failed" }>({ kind: "loading" });
  let login: UserManager | undefined;

  async function accessToken(): Promise<string> {
    const user = await login?.getUser();
    if (!user || user.expired) {
      throw new LoggedOut();
    }

    return user.access_token;
  }

  async function loggedOut() {
    await login?.removeUser();
    session = { kind: "loggedOut" };
  }

  setLogin({
    get name() {
      return session.kind === "loggedIn" ? session.name : "";
    },
    accessToken,
    onLoggedOut: loggedOut,
    onNotAllowed() {
      session = { kind: "notAllowed" };
    },
  });

  onMount(async () => {
    try {
      const config = await api.config();
      if (!config.ok) {
        throw new Error("frogly can't say where to log in");
      }

      login = new UserManager(loginSettings(config.value, window.location.origin));
      session = await resume(login, window.location.pathname, (token) => api.me(async () => token));

      // automaticSilentRenew failed, maybe because the refresh token expired
      login.events.addSilentRenewError(() => {
        // the login page would hide why the person can't get in
        if (session.kind !== "notAllowed") {
          session = { kind: "loggedOut" };
        }
      });
    } catch (error) {
      console.error(error);
      session = { kind: "failed" };
    } finally {
      // the callback URL holds a one-time code, a reload mustn't use it
      // again, whether the login worked or not
      if (window.location.pathname === callbackPath) {
        window.history.replaceState(null, "", "/");
      }
    }
  });
</script>

<header>
  <a class="mark" href={p("/")}><span aria-hidden="true">蛙</span>frogly</a>
  {#if session.kind === "loggedIn"}
    <span class="who">{session.name}</span>
    <Button onclick={() => login?.signoutRedirect()}>log out</Button>
  {/if}
</header>

<main>
  {#if session.kind === "loading"}
    <Loading />
  {:else if session.kind === "failed"}
    <Page heading="frogly can't be reached right now.">
      <p class="quiet">Try again in a moment.</p>
    </Page>
  {:else if session.kind === "loggedIn"}
    <Router />
  {:else if session.kind === "notAllowed"}
    <Page heading="This account can't use frogly.">
      <p class="quiet">Ask whoever runs frogly to let you in, or log in with another account.</p>
      <Button onclick={() => login?.signoutRedirect()}>log out</Button>
    </Page>
  {:else}
    <Page heading="A quiet place for your team's work.">
      {#if session.loginFailed}
        <p class="quiet">The login didn't work. Try again.</p>
      {/if}
      <Button pond onclick={() => login?.signinRedirect()}>log in</Button>
    </Page>
  {/if}
</main>

<style>
  header {
    display: flex;
    align-items: center;
    gap: 18px;
    max-width: 1080px;
    margin: 0 auto;
    padding: 22px 0;
    color: var(--ink-2);
    font-size: 14px;
  }

  .mark {
    font-family: var(--serif);
    font-size: 19px;
    color: var(--ink);
    text-decoration: none;
    letter-spacing: 0.01em;
    margin-right: auto;
  }

  .mark span {
    color: var(--pond);
    margin-right: 6px;
  }

  main {
    max-width: 620px;
    margin: 10vh auto 0;
  }

  @media (max-width: 640px) {
    .who {
      display: none;
    }

    main {
      margin-top: 4vh;
    }
  }
</style>
