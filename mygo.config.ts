import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "roost",
  identifier: "com.ernzst.roost",
  version: "0.2.0",
  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5287",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
  // Installed copies update themselves from the GitHub releases of this
  // repository. The secret key that signs them is in
  // ~/Library/Application Support/mygo/update-keys, outside the repository.
  updates: {
    publicKey: "/D6mD5qdtEtOJaO0DohUAOLJptHgm7PP8wdH0qyXBtM=",
    github: "Ernxst/roost",
    privateKey: "~/Library/Application Support/mygo/update-keys/mygo-update.key",
  },
});
