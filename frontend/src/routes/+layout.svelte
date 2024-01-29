<script>
  import { Toast, initializeStores } from "@skeletonlabs/skeleton";
  import { AppBar, LightSwitch } from "@skeletonlabs/skeleton";
  import { goto } from "$app/navigation";
  import { title } from "$lib/index.js";
  import "../app.postcss";
  let titleString;
  initializeStores();
  title.subscribe((t) => {
    titleString = t;
  });
</script>

<Toast />
<AppBar
  class="mb-4"
  gridColumns="grid-cols-3"
  slotDefault="place-self-center"
  slotTrail="place-content-end"
>
  <svelte:fragment slot="lead">
    <a href="/">WRC-Pacenotes Editor</a></svelte:fragment
  >
  {titleString}
  <svelte:fragment slot="trail"><LightSwitch /></svelte:fragment>
</AppBar>
<slot />
