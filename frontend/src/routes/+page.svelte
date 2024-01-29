<script>
  import { title } from "$lib/index.js";
  export let data;
  title.set("Stage List");
</script>

<svelte:head>
  <title>Stage List</title>
</svelte:head>

<div class="container h-full mx-auto">
  {#each data.Locations as location}
    <details>
      <summary class="cursor-pointer">{location.Name}</summary>
      <ul class="indent-4">
        {#each location.Stages as stage}
          <a
            class="anchor"
            href="/edit/?location={stage.ID.Location}&stage={stage.ID.Stage}"
            ><li>{stage.ID.Stage}.{stage.Stage}</li></a
          >
        {/each}
      </ul>
    </details>
  {/each}
</div>
