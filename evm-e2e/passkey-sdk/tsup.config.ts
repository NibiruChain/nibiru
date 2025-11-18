import { defineConfig } from "tsup"

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm", "cjs"],
  dts: true,
  clean: true,
  esbuildOptions: (opt) => {
    opt.external = opt.external ?? []
    opt.external.push("cbor-web")
  },
})
