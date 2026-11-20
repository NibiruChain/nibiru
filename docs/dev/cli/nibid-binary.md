---
order: 2
footer:
  newsletter: false
description: >-
  Install the Nibiru CLI to query chain data, send transactions, and run nodes.
  Use Bun or npm, download a release binary, or build from source.
---

# Install the Nibiru CLI

{{ $frontmatter.description }}

<!-- toc -->
- [Option 1: Install with Bun or npm](#option-1-install-with-bun-or-npm)
- [Option 2: Use the Bash installer](#option-2-use-the-bash-installer)
- [Option 3: Download a release binary](#option-3-download-a-release-binary)
- [Option 4: Build from source](#option-4-build-from-source)
  - [Install build tools](#install-build-tools)
  - [Install Go](#install-go)
  - [Compile the source code](#compile-the-source-code)
- [Post-installation](#post-installation)
- [Local development](#local-development)
- [Docker Engine](#docker-engine)
- [Next steps](#next-steps)
<!-- tocstop -->

## Option 1: Install with Bun or npm

Install the CLI globally with [Bun](https://bun.sh/docs/installation):

```bash
bun install --global @nibiruchain/nibiru
nibiru version
```

Or use npm:

```bash
npm install --global @nibiruchain/nibiru
nibiru version
```

These commands install the npm default version. The package supports Linux and
macOS on x64 and arm64. On Windows, use WSL. Keep optional dependencies enabled
so the package manager installs the binary for your platform.

The launcher requires [Node.js](https://nodejs.org/en/download) on your `PATH`,
including when you install with Bun. It exposes the command `nibiru`, which runs
the native `nibid` binary of a released version and forwards all arguments.
Replace `nibid` with `nibiru` when following the other CLI guides. The `nibid`
binary itself doesn't answer to the name `nibiru`, so release archives and
source builds use `nibid`.

## Option 2: Use the Bash installer

```bash
curl -s https://get.nibiru.fi/! | bash
```

To install a specific version:

```bash
curl -s https://get.nibiru.fi/@v2.19.0! | bash
```

The `!` suffix moves the binary to `/usr/local/bin` and may prompt for your
`sudo` password. Omit `!` to download without moving the binary.

## Option 3: Download a release binary

Download an archive from the [Nibiru releases](https://github.com/NibiruChain/nibiru/releases)
page. Expand the release's assets to find the download links. Choose `darwin`
for macOS or `linux` for Linux and WSL.

Check your CPU architecture:

```bash
uname -m
```

Use an `amd64` archive for `x86_64`, or an `arm64` archive for `arm64` or `aarch64`.

![Release download assets](../../img/release-assets.png)

For example, extract the Linux amd64 archive for version `v2.19.0`:

```bash
tar -xzf nibid_2.19.0_linux_amd64.tar.gz
```

Add the directory containing `nibid` to your shell configuration, replacing
`/path/to/nibid-directory` with its actual path:

```bash
export PATH="/path/to/nibid-directory:$PATH"
```

Or install the binary in `/usr/local/bin`:

```bash
sudo install -m 755 nibid /usr/local/bin/nibid
```

## Option 4: Build from source

A source build writes the binary `nibiru` and a `nibid` symlink that points to
it. Both names run the same program.

### Install build tools

Install Git, a C compiler, and [just](https://github.com/casey/just#installation).
On macOS, install the Command Line Tools and `just`:

```bash
xcode-select --install
brew install just
```

On Ubuntu or WSL, install Git and the compiler tools, then install `just`
using its installation guide:

```bash
sudo apt-get update
sudo apt-get install --yes git build-essential
```

### Install Go

Install Go with [Homebrew](https://brew.sh):

```bash
brew install go
go version
```

For other installation methods, follow the [official Go installation instructions](https://go.dev/doc/install).
Use a Go version that satisfies the selected release's `go.mod` requirement.

### Compile the source code

Clone the repository, select the release, and install `nibiru` and the `nibid` symlink:

```bash
git clone https://github.com/NibiruChain/nibiru
cd nibiru
just install
```

## Post-installation

For Bun or npm installs, check the version and available commands:

```bash
nibiru version
nibiru --help
```

For the Bash installer, manual download, or source build, use:

```bash
nibid version
nibid --help
```

A source build also installs `nibiru`. If a Bun or npm install of `nibiru` comes
first on your `PATH`, that command runs the released binary. Run
`$(go env GOPATH)/bin/nibiru` to use your build.

If your shell cannot find the command, check the installation's bin directory:

- Bun: run `bun pm bin --global` and add that directory to your `PATH`.
- npm: run `npm prefix --global` and add the returned directory's `bin` subdirectory to your `PATH`.
- Bash installer: check `/usr/local/bin` if you used the `!` suffix.
- Manual download: add the directory where you extracted `nibid`.
- Source build: add Go's bin directory as described in the tip below.

::: tip
If you see `nibid: command not found` after building from source, add Go's bin
directory to your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Save this line in your shell configuration and reload it.
:::

## Which `nibiru` Am I Running?

Several installs can put a `nibiru` command on your `PATH`, for example a Bun
or npm launcher and a source build. Run `nibiru version --long` to see which
file runs. Both the text (YAML) and the JSON (`-o json`) output hold one field,
`binary_paths`, with the same structure. In YAML it sorts first, because the
fields sort by name. Field `running_from` is the running file. Each entry of
`commands` lists every `matches` file on `PATH` in order, marks the running one
with `is_running`, and sets `first_is_running` to `false` when another file wins
the name. When that happens, the command also prints a one-line note to stderr.

```yaml
binary_paths:
  commands:
  - first_is_running: false
    matches:
    - is_running: false
      path: /home/me/.bun/bin/nibiru
      resolved: /home/me/node_modules/@nibiruchain/nibiru/bin/nibiru.js
    - is_running: true
      path: /home/me/go/bin/nibiru
      resolved: /home/me/go/bin/nibiru
    name: nibiru
  running_from: /home/me/go/bin/nibiru
build_deps: ...
```

### Rules for Agents and Scripts

- Plain `nibiru` may be a packaged release, such as a Bun or npm launcher that
  comes first on `PATH`. It is not proof that you run your source build.
- Call a source build by full path: `"$(go env GOPATH)/bin/nibiru"`. If you set
  `GOBIN`, use `"$(go env GOBIN)/nibiru"` instead.
- Before a step that depends on the version, run `nibiru version --long -o json`
  and read field `binary_paths`. Field `running_from` is the file that runs.
  For each entry of `commands`, `first_is_running` is `false` when another file
  with that name wins on `PATH`.

```bash
~/go/bin/nibiru version --long -o json | jq '.binary_paths.commands[0] | {name, first_is_running}'
```

```json
{
  "name": "nibiru",
  "first_is_running": false
}
```

The script `contrib/scripts/build-nibiru.sh` (run by `just install`) prints
one marker line per name when it finishes. Each line is plain text with
`key=value` fields, so a script can match it:

- `nibiru-path-shadowed name=<nibiru|nibid> runs=<path> installed=<path>`: the
  name resolves to another file on `PATH`. Field `runs` is that file and
  `installed` is the build just written. Call `installed` by full path or put
  its directory first on `PATH`.
- `nibiru-path-ok name=<nibiru|nibid> runs=<path> other=<path>`: the installed
  build wins, and a different file with the same name also sits on `PATH`.
  Field `other` is that file.

The installer prints no marker when only one copy exists, and it exits `0` in
every case. The `nibid` symlink that points to the installed file never counts
as another copy.

## Local development

Print and run the embedded single-node localnet script:

```bash
nibid localnet --script | bash
```

For Bun or npm installs, use:

```bash
nibiru localnet --script | BINARY=nibiru bash
```

Setting `BINARY=nibiru` makes the script use the npm launcher for node commands.
Stop any running localnet before running the script. It resets local chain data
under `$HOME/.nibid` before starting the node. Open another terminal to query the
local chain or send transactions.

## Command List

Run `nibid --help` to list the commands: `tx`, `q` (alias `query`), `keys`,
`config`, `status`, `debug`, `localnet`, `node`, `help`, and `version`.
Command `node` holds `start`, `init`, `export`, `genesis`, `prune`,
`rollback`, `bft` (alias `tendermint`), `testnet`, and `evm-tx-index`.

Add `--help-verbose` to any command to list the log and trace flags that help
hides by default.

The old top-level node commands, such as `nibid start` and
`nibid tendermint show-node-id`, keep working and print no warning. They no
longer appear in `nibid --help`. Cosmovisor, Docker images, and systemd units
that call `nibid start` need no change.

## Docker Engine

Some repository workflows use Docker containers. Follow the
[Docker Engine installation instructions](https://docs.docker.com/engine/install/)
when working with those workflows.

---

## Next steps

- [Use the Nibiru CLI][page-cli]
- [Set up Cosmovisor][page-cosmovisor]
- [Run a full node][page-full-node]
- [Set up a validator][page-validator]
- [Learn about the node daemon][page-node-daemon]

[page-cosmovisor]: ../../run-nodes/full-nodes/cosmovisor.md
[page-full-node]: ../../run-nodes/full-nodes/index.md
[page-validator]: ../../run-nodes/validators
[page-node-daemon]: ../../run-nodes/full-nodes/node-daemon.md
[page-cli]: ./
