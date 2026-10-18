<script lang="ts">
  import { showTitle } from "../lib/title";

  // title names the tab already, before the page is there to do it
  let { title }: { title?: string } = $props();

  $effect(() => {
    if (title !== undefined) {
      showTitle(title);
    }
  });
</script>

<p class="quiet" aria-busy="true">…</p>
