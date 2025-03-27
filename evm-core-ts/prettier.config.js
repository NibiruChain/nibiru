// prettier.config.js, .prettierrc.js, prettier.config.mjs, or .prettierrc.mjs

/** @type {import("prettier").Config} */
const config = {
  trailingComma: "all",
  tabWidth: 2,
  printWidth: 80,
  semi: false,
  singleQuote: false,
  arrowParens: "always",

  plugins: ["@ianvs/prettier-plugin-sort-imports"],
  /** Using the "@ianvs/prettier-plugin-sort-imports" plugin, the `importOrder` field
   * controls the order in which imports are sorted.
   *
   * 1. "<BUILT_IN_MODULES>" - Built-in Node.js modules like "fs", "path", and
   * "http" that don't require an npm installation.
   *
   * 2. "<THIRD_PARTY_MODULES>" - Module like "react", "lodash", or "bun"
   *
   * 3. "^@/(.*)$" - For paths beginning with "@"
   * 4. "^~/(.*)$" - For paths beginning with "~"
   * 5. "^[.]" - Relative imports like "./file" or "../state/utils".
   * */
  importOrder: [
    "<BUILT_IN_MODULES>",
    "<THIRD_PARTY_MODULES>",
    "", // creates a blank line in the import block
    "^@/(.*)$",
    "^~/(.*)$",
    "", // creates a blank line in the import block
    "^[.]",
  ],

}

export default config
