<script lang="ts">
  import { onMount, type Snippet } from "svelte";

  import { showTitle } from "../lib/title";

  // title names the page in the tab, or just frogly without it. heading is
  // what it says on top, and what a screen reader announces once the page
  // shows.
  let { title, heading, children }: { title?: string; heading: string; children?: Snippet } = $props();

  let top = $state<HTMLHeadingElement>();

  $effect(() => {
    showTitle(title);
  });

  // the router swaps pages without loading a new one, so the focus would
  // stay on the link that was clicked, somewhere on the old page
  onMount(() => {
    top?.focus({ preventScroll: true });
  });
</script>

<h1 tabindex="-1" bind:this={top}>{heading}</h1>
{@render children?.()}

<style>
  h1:focus {
    outline: none;
  }
</style>
