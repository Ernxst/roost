import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

const fixtureRuntime = fileURLToPath(new URL("./src/fixture-runtime.ts", import.meta.url));

export default defineConfig({
  plugins: [
    {
      // The page's imports of the runtime go through the fixture switch, which
      // imports the runtime itself.
      name: "fixture-runtime",
      apply: "serve",
      enforce: "pre",
      resolveId(id, importer) {
        if (id === "mygo-runtime" && importer?.split("?")[0] !== fixtureRuntime) return fixtureRuntime;
      },
    },
  ],
  server: {
    // The app loads this address during `mygo dev` (devUrl in mygo.json).
    port: 5287,
    strictPort: true,
    // The development app and the packaged builds are not part of the page,
    // so changes to them should not reload it.
    watch: { ignored: ["**/.mygo/**", "**/build/**"] },
  },
});
