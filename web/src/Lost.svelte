<script lang="ts">
  import { p } from "./router";
  import Page from "./ui/Page.svelte";
</script>

<Page title="Nothing here" heading="There's nothing here.">
  <p class="quiet"><a href={p("/")}>Back to your projects</a></p>
</Page>
