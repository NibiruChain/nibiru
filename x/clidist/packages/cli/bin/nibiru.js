#!/usr/bin/env node

"use strict";

const { spawn } = require("node:child_process");
const { dirname, join } = require("node:path");

const packages = {
  "linux-x64": "@nibiruchain/nibiru-linux-x64",
  "linux-arm64": "@nibiruchain/nibiru-linux-arm64",
  "darwin-x64": "@nibiruchain/nibiru-darwin-x64",
  "darwin-arm64": "@nibiruchain/nibiru-darwin-arm64",
};

const target = `${process.platform}-${process.arch}`;
const packageName = packages[target];

if (!packageName) {
  console.error(`nibiru does not support ${target}. Supported targets: ${Object.keys(packages).join(", ")}.`);
  process.exit(1);
}

let packageJson;
try {
  packageJson = require.resolve(`${packageName}/package.json`);
} catch {
  console.error(`nibiru's native package for ${target} is missing (${packageName}). Reinstall without --omit=optional.`);
  process.exit(1);
}

const binary = join(dirname(packageJson), "bin", "nibid");
const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });

child.on("error", (error) => {
  console.error(`failed to start ${binary}: ${error.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
