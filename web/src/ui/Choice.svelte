<script lang="ts" generics="T extends string">
  // radios are never disabled while saving, a disabled radio would lose the
  // focus. The action refuses a second save itself.
  let {
    legend,
    name,
    choices,
    group = $bindable(),
    describedby,
  }: {
    legend: string;
    name: string;
    choices: { value: T; text: string }[];
    group?: T;
    describedby?: string;
  } = $props();
</script>

<fieldset aria-describedby={describedby}>
  <legend class="label">{legend}</legend>
  {#each choices as choice (choice.value)}
    <label><input type="radio" {name} value={choice.value} bind:group /> {choice.text}</label>
  {/each}
</fieldset>

<style>
  fieldset {
    border: 0;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  legend {
    padding: 0;
    margin-bottom: 10px;
  }

  input {
    accent-color: var(--pond);
  }
</style>
