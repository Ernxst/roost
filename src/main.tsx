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

render(<App />, document.getElementById("app")!);
