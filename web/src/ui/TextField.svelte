<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  import Problem from "./Problem.svelte";

  // label is for screen readers, the placeholder says it to everyone else.
  // Without it, the page labels the field itself. busy makes the field
  // readonly, not disabled, since a disabled field would lose the focus.
  // A field that can have a problem shows it below, unless problemId says
  // where the page shows it.
  let {
    id,
    label,
    value = $bindable(""),
    busy = false,
    problem,
    problemId,
    ref = $bindable(),
    "aria-describedby": describedby,
    ...rest
  }: HTMLInputAttributes & {
    id: string;
    label?: string;
    value?: string;
    busy?: boolean;
    problem?: string;
    problemId?: string;
    ref?: HTMLInputElement;
  } = $props();

  const ownProblem = $derived(problem !== undefined && problemId === undefined);
  const describing = $derived(
    [describedby, problemId, ownProblem ? `${id}-problem` : undefined].filter(Boolean).join(" ") || undefined,
  );
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
  aria-describedby={describing}
  {...rest}
/>
{#if ownProblem}
  <Problem id={`${id}-problem`} text={problem ?? ""} />
{/if}

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
