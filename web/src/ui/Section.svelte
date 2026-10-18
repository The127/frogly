<script lang="ts">
  import type { Snippet } from "svelte";

  // label is the section's heading. top is where an action can move the
  // focus when the control that had it is gone, like a list row.
  let { label, id, top = $bindable(), children }: { label: string; id: string; top?: HTMLElement; children: Snippet } = $props();
</script>

<section aria-labelledby={id}>
  <h2 class="label" {id} tabindex="-1" bind:this={top}>{label}</h2>
  {@render children()}
</section>

<style>
  section {
    margin-top: 44px;
  }

  h2 {
    margin: 0 0 10px;
  }

  h2:focus {
    outline: none;
  }
</style>
