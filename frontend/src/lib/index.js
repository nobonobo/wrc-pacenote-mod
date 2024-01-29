// place files you want to import through the `$lib` alias in this folder.
import { writable } from "svelte/store";

function createTitle() {
  const { subscribe, set, update } = writable(0);
  return {
    subscribe: subscribe,
    set: set,
    update: update,
  };
}

export const title = createTitle();
