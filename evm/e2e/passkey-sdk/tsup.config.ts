import { defineConfig } from "tsup"

export default defineConfig({
  entry: ["src/index.ts", "src/passkey-e2e.ts", "src/local-bundler.ts"],
  format: ["esm", "cjs"],
  dts: true,
  clean: true,
  esbuildOptions: (opt) => {
    opt.external = opt.external ?? []
    opt.external.push("cbor-web")
  },
})
