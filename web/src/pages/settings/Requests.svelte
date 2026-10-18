<script lang="ts">
  import { Action, type Resource } from "../../lib/action.svelte";
  import * as api from "../../lib/api";
  import { getLogin } from "../../lib/login";
  import Button from "../../ui/Button.svelte";
  import List from "../../ui/List.svelte";
  import Notice from "../../ui/Notice.svelte";
  import Problem from "../../ui/Problem.svelte";
  import Row from "../../ui/Row.svelte";
  import Rows from "../../ui/Rows.svelte";
  import Section from "../../ui/Section.svelte";

  let { projectId, requests, gone }: { projectId: string; requests: Resource<api.WaitingRequest[]>; gone: () => void } = $props();

  const login = getLogin();
  const answering = new Action(login, () => gone());
  let top = $state<HTMLElement>();

  function answer(request: api.WaitingRequest, reply: "yes" | "no") {
    const send = reply === "yes" ? api.accept : api.decline;

    // the request is gone, answered by another member at the same moment
    // or since, and the list shows what's left
    const answeredAlready = async () => {
      answering.problem = `${request.name}'s request was answered already.`;
      await requests.load();
    };

    return answering.run(() => send(login.accessToken, projectId, request.id), {
      what: `answer ${request.name}`,
      done: () => {
        requests.change((waiting) => waiting.filter((other) => other.id !== request.id));
        // the button that had the focus is gone with the request
        top?.focus();

        return reply === "yes" ? `${request.name} is a member now.` : `You said no to ${request.name}.`;
      },
      notFound: answeredAlready,
      conflict: answeredAlready,
    });
  }
</script>

<Section label="asked to join" id="requests-label" bind:top>
  <List
    resource={requests}
    failed="frogly can't show who asked to join right now. Reload the page in a moment."
    empty="Nobody waits for an answer right now."
  >
    {#snippet children(waiting)}
      <Rows>
        {#each waiting as request (request.id)}
          <Row meta={request.email}>
            {request.name}
            {#snippet actions()}
              <Button pond busy={answering.busy} onclick={() => answer(request, "yes")}>yes <span class="hidden">to {request.name}</span></Button>
              <Button busy={answering.busy} onclick={() => answer(request, "no")}>no <span class="hidden">to {request.name}</span></Button>
            {/snippet}
          </Row>
        {/each}
      </Rows>
    {/snippet}
  </List>
  <Problem text={answering.problem} />
  <Notice text={answering.notice} />
</Section>
