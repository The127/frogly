<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  // label is for screen readers, the placeholder says it to everyone else.
  // Without it, the page labels the field itself.
  // busy makes the field readonly, not disabled, since a disabled field
  // would lose the focus.
  let {
    id,
    label,
    value = $bindable(""),
    busy = false,
    problem = "",
    describedby,
    ref = $bindable(),
    ...rest
  }: HTMLInputAttributes & {
    id: string;
    label?: string;
    value?: string;
    busy?: boolean;
    problem?: string;
    describedby?: string;
    ref?: HTMLInputElement;
  } = $props();
</script>

{#if label}
  <label class="hidden" for={id}>{label}</label>
{/if}
<input
  {id}
  bind:value
  bind:this={ref}
  readonly={busy}
  aria-busy={busy}
  aria-invalid={problem ? "true" : undefined}
  aria-describedby={describedby}
  {...rest}
/>

<style>
  input {
    width: 100%;
    border: 0;
    border-bottom: 1px solid var(--line);
    background: transparent;
    color: var(--ink);
    font: 22px/1.4 var(--serif);
    padding: 14px 2px;
    border-radius: 0;
  }

  input::placeholder {
    color: var(--ink-3);
  }

  input:focus {
    outline: none;
    border-bottom-color: var(--pond);
  }

  input:focus-visible {
    outline: 2px solid var(--pond);
    outline-offset: 2px;
  }
</style>
