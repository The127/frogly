<script lang="ts">
  // polite, so a screen reader says what worked without breaking in
  let { text }: { text: string } = $props();
</script>

<p class="quiet" aria-live="polite">{text}</p>
