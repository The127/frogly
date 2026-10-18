<script lang="ts">
  import type { Snippet } from "svelte";

  // a row of a list: what it is, a quiet line about it, and what a person
  // can do with it
  let { meta, actions, serif = false, children }: { meta?: string; actions?: Snippet; serif?: boolean; children: Snippet } = $props();
</script>

<li>
  <span class="what">
    <span class:serif>{@render children()}</span>
    {#if meta}
      <span class="meta">{meta}</span>
    {/if}
  </span>
  {#if actions}
    <span class="acts">{@render actions()}</span>
  {/if}
</li>

<style>
  li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    padding: 12px 0;
    border-bottom: 1px solid var(--line);
  }

  .what {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .serif {
    font-family: var(--serif);
    font-size: 17px;
  }

  .meta {
    display: block;
    font-size: 13px;
    color: var(--ink-3);
  }

  .acts {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-end;
  }
</style>
