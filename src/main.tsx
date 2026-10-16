import { onPreferences, preferences, type ThemePreferences } from "mygo-runtime";
import { render } from "preact";
import { App } from "./app";
import "./style.css";

// A change of appearance restyles nearly every element at once; let it snap
// rather than run every colour transition together.
matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
  const still = document.createElement("style");
  still.textContent = "*, *::before, *::after { transition: none !important; }";
  document.head.append(still);
  void document.body.offsetHeight;
  requestAnimationFrame(() => still.remove());
});

// Reduce Transparency turns the rail's material into plain paper.
const applyPreferences = ({ reduceTransparency }: ThemePreferences) =>
  document.documentElement.classList.toggle("opaque", reduceTransparency);
// Outside a mygo window, as under `bun run dev:web`, there is no desktop to ask.
if ("mygo" in globalThis) {
  applyPreferences(preferences());
  onPreferences(applyPreferences);
}

render(<App />, document.getElementById("app")!);
