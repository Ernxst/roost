export type FixtureMode = "demo" | "sample" | "worst" | "empty" | "one" | "thousand";

const MODES: FixtureMode[] = ["demo", "sample", "worst", "empty", "one", "thousand"];
const requestedMode = new URLSearchParams(window.location.search).get("data");
export const fixtureMode: FixtureMode = import.meta.env.DEV && MODES.includes(requestedMode as FixtureMode)
  ? (requestedMode as FixtureMode)
  : "demo";

export function fixtureHref(mode: FixtureMode): string {
  const url = new URL(window.location.href);
  if (mode === "demo") url.searchParams.delete("data");
  else url.searchParams.set("data", mode);
  return `${url.pathname}${url.search}${url.hash}`;
}
