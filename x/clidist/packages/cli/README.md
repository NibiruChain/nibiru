# @nibiruchain/nibiru

The Nibiru command-line client for nodes, validators, and developers.

## Install

```bash
bun add --global @nibiruchain/nibiru
nibiru version
```

The package also works with npm:

```bash
npm install --global @nibiruchain/nibiru
nibiru version
```

`nibiru` is available on Linux and macOS for x64 and arm64. npm installs the
matching native package for the host platform. It does not download a binary in
an install script.

## Commands and docs

The npm command `nibiru` launches Nibiru's native `nibid` binary and passes all
arguments through unchanged. Existing `nibid` command documentation therefore
applies to the npm install. See the [Nibiru CLI documentation](https://nibiru.fi/docs/dev/cli)
for node setup, queries, transactions, and configuration.

If npm omits the native package, reinstall without `--omit=optional`.
