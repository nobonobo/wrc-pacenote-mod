export async function load({ fetch }) {
  return {
    Locations: await (await fetch("/api/locations")).json(),
  };
}
