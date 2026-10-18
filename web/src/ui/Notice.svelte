<script lang="ts">
  // polite, so a screen reader says what worked without breaking in. tight
  // drops the space around it, for a notice next to a button.
  let { text, tight = false }: { text: string; tight?: boolean } = $props();
</script>

<p class="quiet" class:tight aria-live="polite">{text}</p>

<style>
  .tight {
    margin: 0;
  }
</style>
