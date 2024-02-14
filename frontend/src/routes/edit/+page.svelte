<script>
  import { onMount } from "svelte";
  import { beforeNavigate } from "$app/navigation";
  import { title } from "$lib/index.js";
  export let data;
  import WaveSurfer from "wavesurfer.js";
  import RegionsPlugin from "$lib/../../node_modules/wavesurfer.js/dist/plugins/regions.esm.js";
  import SvgPanZoom from "svg-pan-zoom";
  import { getToastStore } from "@skeletonlabs/skeleton";
  const toastStore = getToastStore();
  let loop = true;
  let zoom = 100;
  let activeRegion = null;
  let ws = null;
  let submit = null;
  let lastTick = 0;
  let lastIndex = 0;
  let saved = true;
  function beforeUnload(ev) {
    if (!saved) return "exit?";
  }
  beforeNavigate((nav) => {
    if (!saved) {
      nav.cancel();
      toastStore.trigger({
        message: "Regions unsaved!",
        background: "variant-filled-error",
      });
    }
  });
  onMount(async () => {
    title.set(document.title);
    const w = WaveSurfer.create({
      container: "#waveform",
      waveColor: "#4F4A85",
      progressColor: "#383351",
      normalize: true,
      mediaControls: true,
      autoplay: false,
      url: "/api/files/" + data.url + "capture.wav",
    });
    ws = w;
    const wsRegions = ws.registerPlugin(RegionsPlugin.create());
    wsRegions.enableDragSelection({
      color: "rgba(255, 0, 0, 0.1)",
      content: "unknown",
      contentEditable: true,
    });
    ws.on("seeking", () => {
      lastTick = 0;
      lastIndex = 0;
    });
    ws.on("click", () => {
      console.log("ws clicked");
      if (activeRegion) {
        activeRegion.setOptions({ color: "rgba(255, 0, 0, 0.1)" });
        activeRegion.content.blur();
        activeRegion = null;
      }
    });
    ws.on("timeupdate", (currentTime) => {
      let tick = Math.trunc(currentTime * 1000000000);
      let svgDocument = document.getElementById("map").getSVGDocument();
      let parent = svgDocument.getElementById("points");
      let vehicle = svgDocument.getElementById("vehicle").children[0];
      for (let i = lastIndex; i < parent.children.length; i++) {
        let g = parent.children[i];
        lastTick = Number(g.id);
        if (lastTick > tick) {
          lastIndex = i;
          vehicle.cx.baseVal.value = g.children[0].x1.baseVal.value;
          vehicle.cy.baseVal.value = g.children[0].y1.baseVal.value;
          break;
        }
      }
    });
    ws.on("decode", () => {
      ws.zoom(zoom);

      wsRegions.on("region-clicked", (region, e) => {
        e.stopPropagation(); // prevent triggering a click on the waveform
        if (activeRegion) {
          activeRegion.setOptions({ color: "rgba(255, 0, 0, 0.1)" });
          if (region != activeRegion) activeRegion.content.blur();
        }
        activeRegion = region;
        if (activeRegion)
          activeRegion.setOptions({ color: "rgba(0, 255, 0, 0.1)" });
      });
      wsRegions.on("region-double-clicked", (region, e) => {
        e.stopPropagation(); // prevent triggering a click on the waveform
        console.log("double-clicked");
        if (activeRegion != null) {
          if (!ws.isPlaying()) activeRegion.play();
          else ws.pause();
        }
      });
      wsRegions.on("region-out", (region) => {
        if (activeRegion === region) {
          if (loop) {
            region.play();
          } else {
            activeRegion = null;
          }
        }
      });
      let modified = () => {
        saved = false;
      };
      wsRegions.on("region-created", modified);
      wsRegions.on("region-updated", modified);
      wsRegions.on("region-removed", modified);
    });
    ws.on("ready", () => {
      data.regions.filter((r) => {
        let region = wsRegions.addRegion({
          start: r.start,
          end: r.end,
          color: "rgba(255, 0, 0, 0.1)",
          content: r.content,
          contentEditable: true,
        });
        saved = true;
      });
    });
    submit = async (ev) => {
      let res = [];
      wsRegions.getRegions().filter((r) => {
        res.push({
          start: r.start,
          end: r.end,
          content: r.content.innerText,
        });
      });
      try {
        let result = await (
          await fetch("/api/regions/" + data.url, {
            method: "POST",
            headers: {
              Accept: "application/json",
              "Content-Type": "application/json",
            },
            body: JSON.stringify(res),
          })
        ).json();
        if (result.success) {
          toastStore.trigger({
            message: "Regions save successfull!",
            background: "variant-filled-success",
          });
          saved = true;
        } else {
          toastStore.trigger({
            message: "Regions save failed!",
            background: "variant-filled-error",
          });
        }
      } catch (e) {
        toastStore.trigger({
          message: "Regions save failed!",
          background: "variant-filled-error",
        });
      }
    };
  });
  function getEditting() {
    if (activeRegion == null) return null;
    if (activeRegion.element == null) return null;
    return activeRegion.element.querySelector(
      "div[contenteditable=true]:focus"
    );
  }
  window.getEditting = getEditting;
  function isEditting() {
    return getEditting() != null;
  }
  async function keyDown(ev) {
    console.log("key:", ev.keyCode);
    if (ev.keyCode == 27) {
      if (activeRegion != null) activeRegion.content.blur();
    } else if (ev.keyCode == 13) {
      if (activeRegion != null) {
        if (isEditting()) {
          activeRegion.content.blur();
        } else {
          ev.preventDefault();
          await fetch("/api/speech", {
            method: "POST",
            Headers: {
              Accept: "application/json",
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ text: activeRegion.content.innerText }),
          });
        }
      }
    } else if (ev.keyCode == 32) {
      if (activeRegion == null || !isEditting()) {
        ev.preventDefault(); // cancel browser scrolling
        ws.playPause();
      }
    } else if (ev.keyCode == 46) {
      if (activeRegion && !isEditting()) {
        ws.pause();
        activeRegion.remove();
        activeRegion = null;
      }
    }
  }
  function loaded(ev) {
    SvgPanZoom("#map", {
      zoomEnabled: true,
      controlIconsEnabled: true,
    });
  }
</script>

<svelte:window on:keydown={keyDown} on:beforeunload={beforeUnload} />
<svelte:head>
  <title>{data.stage}</title>
</svelte:head>

<div class="container h-full flex flex-col gap-y-6 mx-auto">
  <div id="waveform"></div>
  <div class="flex gap-x-8">
    <label class="flex-none h-8">
      Loop regions:<input
        type="checkbox"
        class="checkbox block"
        bind:checked={loop}
      />
    </label>
    <label class="flex-auto h-8">
      Zoom: <input
        type="range"
        class="range block"
        min="10"
        max="1000"
        value={zoom}
        on:input={(e) => {
          ws.zoom(Number(e.target.value));
        }}
      />
    </label>
    <div class="flex-none h-8">
      <button
        class="btn variant-soft-primary"
        disabled={submit == null || saved}
        on:click={submit}>Save</button
      >
    </div>
  </div>
  <div class="border-blue-900 border-2 rounded-lg">
    <embed
      class="w-full"
      style="height: 50vh"
      on:load={loaded}
      id="map"
      type="image/svg+xml"
      src={"/api/map/" + data.url}
    />
  </div>
</div>
