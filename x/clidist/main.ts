import { createProgram } from "./src/nibiru-dist";

await createProgram().parseAsync().catch((error: unknown) => {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
});
