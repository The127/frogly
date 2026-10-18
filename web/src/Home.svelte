<script lang="ts">
  import { getLogin } from "./lib/login";
  import Projects from "./Projects.svelte";
  import PublicProjects from "./PublicProjects.svelte";

  const login = getLogin();
</script>

<h1>Hello, {login.name}</h1>
<Projects />
<PublicProjects />
