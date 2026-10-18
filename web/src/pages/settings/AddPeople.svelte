<script lang="ts">
  import { Action, Busy } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { peopleBehind } from "../../lib/format";
  import { getLogin } from "../../lib/login";
  import Button from "../../ui/Button.svelte";
  import Notice from "../../ui/Notice.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Row from "../../ui/Row.svelte";
  import Rows from "../../ui/Rows.svelte";
  import Section from "../../ui/Section.svelte";
  import TextField from "../../ui/TextField.svelte";

  let { projectId, gone }: { projectId: string; gone: () => void } = $props();

  const login = getLogin();
  // one busy flag for searching and adding, so a search that finishes late
  // can't bring back someone just added
  const busy = new Busy();
  const searching = new Action(login, () => gone(), busy);
  const adding = new Action(login, () => gone(), busy);

  let text = $state("");
  let people = $state<api.Person[]>();
  let field = $state<HTMLInputElement>();

  function search(event: SubmitEvent) {
    event.preventDefault();
    if (busy.now) {
      return;
    }

    adding.problem = "";
    adding.notice = "";
    // empties the count, so the same number found again is announced again
    people = undefined;

    return searching.run(() => api.searchPeople(login.accessToken, projectId, text), {
      what: "search right now",
      done: (found) => {
        people = found;
      },
    });
  }

  function add(person: api.Person) {
    return adding.run(() => api.addMember(login.accessToken, projectId, person.id), {
      what: `add ${person.name}`,
      done: (member) => {
        people = people?.filter((found) => found.id !== person.id);
        // the button that had the focus is gone with the person
        field?.focus();

        return `${member.name} is a member now.`;
      },
    });
  }
</script>

<Section label="add someone" id="people-label">
  <form onsubmit={search}>
    <TextField
      id="people-search"
      label="Name or email of someone who uses frogly"
      bind:value={text}
      bind:ref={field}
      placeholder="find someone by name or email…"
      busy={searching.busy}
      problem={searching.problem}
      describedby="people-problem"
    />
  </form>
  <Problem id="people-problem" text={searching.problem} />

  <!-- polite, so a screen reader hears what the search found -->
  <p class="hidden" aria-live="polite">{people === undefined ? "" : `${peopleBehind(people.length)} found`}</p>
  {#if people !== undefined}
    {#if people.length === 0}
      <p class="quiet">Nobody found who could join.</p>
    {:else}
      <Rows>
        {#each people as person (person.id)}
          <Row meta={person.email}>
            {person.name}
            {#snippet actions()}
              <Button busy={adding.busy} onclick={() => add(person)}>add <span class="hidden">{person.name}</span></Button>
            {/snippet}
          </Row>
        {/each}
      </Rows>
    {/if}
  {/if}
  <Problem text={adding.problem} />
  <Notice text={adding.notice} />
</Section>
