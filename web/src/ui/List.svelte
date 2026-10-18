<script lang="ts" generics="T">
  import type { Snippet } from "svelte";

  import type { Resource } from "../lib/action.svelte";
  import Loading from "./Loading.svelte";

  // List shows a list a page loads: while it loads, when it can't, when
  // it's empty, and when loading it again failed, so it may be old. failed
  // says what can't be shown, List adds what to do about it.
  let {
    resource,
    failed,
    empty,
    children,
  }: {
    resource: Resource<T[]>;
    failed: string;
    empty: string;
    children: Snippet<[T[]]>;
  } = $props();
</script>

{#if resource.state.kind === "loading"}
  <Loading />
{:else if resource.state.kind !== "loaded"}
  <p class="quiet">{failed} Reload the page in a moment.</p>
{:else}
  {#if resource.stale}
    <p class="quiet small">frogly can't show the latest of this list right now.</p>
  {/if}
  {#if resource.state.value.length === 0}
    <p class="quiet">{empty}</p>
  {:else}
    {@render children(resource.state.value)}
  {/if}
{/if}
