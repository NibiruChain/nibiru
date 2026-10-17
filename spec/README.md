# Nibiru repository specifications

This directory contains durable technical records for contributors and
maintainers of this repository. It is part of the public source tree, but it
is not part of the Nibiru website documentation.

## File conventions

- A directory `README.md` explains the code in that directory. Keep local
  setup, commands, and architecture close to the code they describe.
- A numbered `NN-*.md` specification records a topic that crosses directories
  or needs a durable source of truth. Do not renumber an existing file.
- Each specification identifies its status, code scope, and the code, script,
  or workflow that establishes its behavior.

## Initial specifications

- [01. Building Nibiru from source](./01-building-nibiru-from-source.md)
- [02. Chaosnet](./02-chaosnet.md)
- [03. Releasing Nibiru](./03-releasing-nibiru.md)
- [04. Rust contract workspace](./04-rust-contract-workspace.md)
- [05. Publishing coupled Rust crates](./05-publishing-coupled-rust-crates.md)
