<script lang="ts">
  import { p } from "../router";
  import Back from "../ui/Back.svelte";
  import Page from "../ui/Page.svelte";
</script>

<Back href={p("/")} text="your projects" />

<Page title="Nothing here" heading="There's nothing here." />
