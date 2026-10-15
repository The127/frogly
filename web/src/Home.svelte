<script lang="ts">
  import { getLogin } from "./login";
  import Projects from "./Projects.svelte";

  const login = getLogin();
</script>

<h1>Hello, {login.name}</h1>
<Projects accessToken={login.accessToken} onLoggedOut={login.onLoggedOut} />
