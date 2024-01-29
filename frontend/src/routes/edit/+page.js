export async function load({ fetch, url }) {
  let params = url.searchParams;
  let u = params.get("location") + "/" + params.get("stage") + "/";
  let stage = await (await fetch("/api/stage/" + u)).json();
  let regions = await (await fetch("/api/regions/" + u)).json();
  return {
    url: u,
    params: params,
    stage: stage,
    regions: regions,
  };
}
