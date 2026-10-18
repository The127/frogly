<script lang="ts">
  import { getLogin } from "../../lib/login";
  import Page from "../../ui/Page.svelte";
  import PublicProjects from "./PublicProjects.svelte";
  import YourProjects from "./YourProjects.svelte";

  const login = getLogin();
</script>

<Page title="Your projects" heading={`Hello, ${login.name}`}>
  <YourProjects />
  <PublicProjects />
</Page>
