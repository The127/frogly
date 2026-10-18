<script lang="ts" generics="T">
  import type { Snippet } from "svelte";

  import type { Resource } from "../lib/action.svelte";
  import Page from "./Page.svelte";

  // Loaded shows a page's resource: the page once it's there, or what to
  // say while it loads, when it's gone, and when it can't be shown.
  let {
    resource,
    title,
    notFound,
    failed,
    children,
  }: {
    resource: Resource<T>;
    title: string;
    notFound: { heading: string; text: string };
    failed: string;
    children: Snippet<[T]>;
  } = $props();

  // Page names the tab once there's something to show
  $effect(() => {
    if (resource.state.kind === "loading") {
      document.title = `${title} · frogly`;
    }
  });
</script>

{#if resource.state.kind === "loading"}
  <p class="quiet" aria-busy="true">…</p>
{:else if resource.state.kind === "notFound"}
  <Page {title} heading={notFound.heading}>
    <p class="quiet">{notFound.text}</p>
  </Page>
{:else if resource.state.kind === "failed"}
  <Page {title} heading={failed}>
    <p class="quiet">Reload the page in a moment.</p>
  </Page>
{:else}
  {@render children(resource.state.value)}
{/if}
