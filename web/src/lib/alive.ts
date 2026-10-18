import { onDestroy } from "svelte";

// alive says whether the component that called it is still shown, for an
// action that finishes after the person moved on and mustn't pull them
// back. Call it while the component starts.
export function alive(): () => boolean {
  let shown = true;
  onDestroy(() => {
    shown = false;
  });

  return () => shown;
}
