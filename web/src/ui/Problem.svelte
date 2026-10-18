<script lang="ts">
  // always there, so a screen reader announces the text when it changes
  let { text, id }: { text: string; id?: string } = $props();
</script>

<p class="problem" {id} role="alert">{text}</p>

<style>
  .problem {
    color: var(--wait);
    margin: 8px 0 0;
    min-height: 1.6em;
  }
</style>
