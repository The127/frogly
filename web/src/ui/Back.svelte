<script lang="ts">
  let { href, text }: { href: string; text: string } = $props();
</script>

<p class="back"><a {href}><span aria-hidden="true">←</span> {text}</a></p>

<style>
  .back {
    margin: 0 0 18px;
    color: var(--ink-3);
    font-size: 14px;
  }

  a {
    text-decoration: none;
  }

  a:hover,
  a:focus-visible {
    color: var(--pond);
  }
</style>
