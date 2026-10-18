<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLButtonAttributes } from "svelte/elements";

  // busy keeps the button where it is, so the focus stays, and the action
  // refuses a second run itself
  let {
    busy = false,
    pond = false,
    ref = $bindable(),
    children,
    ...rest
  }: HTMLButtonAttributes & { busy?: boolean; pond?: boolean; ref?: HTMLButtonElement; children: Snippet } = $props();
</script>

<button class:pond aria-disabled={busy} bind:this={ref} {...rest}>{@render children()}</button>

<style>
  button {
    font: inherit;
    font-size: 14px;
    border: 1px solid var(--line);
    background: transparent;
    color: var(--ink);
    padding: 7px 14px;
    border-radius: 999px;
    cursor: pointer;
    white-space: nowrap;
  }

  button:hover {
    border-color: var(--ink-3);
  }

  .pond,
  .pond:hover {
    border-color: var(--pond);
    color: var(--pond);
  }

  button:focus-visible {
    outline: 2px solid var(--pond);
    outline-offset: 2px;
  }

  button[aria-disabled="true"] {
    cursor: progress;
    opacity: 0.6;
  }
</style>
