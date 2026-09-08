# NibiruChain/nibiru-wasm

Wasm smart contract sandbox for Nibiru.

```bash
⚡ NibiruChain/nibiru-wasm
├── 📂 artifacts         # compiled .wasm smart contracts for nibiru-wasm
├── 📂 contracts         # Smart contracts for Nibiru
    └── 📂 nibi-stargate # Example contract using nibiru-std for CosmosMsg::Stargate
    └── 📂 incentives    # Generalized incentives over time for locked tokens
    └── 📂 lockup        # For locking and unlocking tokens like LP tokens
    └── 📂 core-cw3-flex-msig # CW3-flex-multisig with stargate enabled.
    └── 📂 core-token-vesting # Token linear vesting contracts with optional cliffs.
    └── 📂 core-token-vesting-v2 # Improved version of core-token-vesting-v2.
├── 📂 nibiru-std      # Nibiru standard library for smart contracts
    └── 📦 proto       # Types and traits for QueryRequest::Stargate and CosmosMsg::Stargate
         └──           #   Includes constructors for Cosmos, IBC, and Nibiru. 
├── 📂 packages        # Other Rust packages
    └── 📦 cw-address-like # Address-like helper traits for CosmWasm types.
    └── 📦 easy-addr       # Address construction helpers for tests.
    └── 📦 nibiru-ownable  # Ownership patterns for contracts.
├── Cargo.toml
├── Cargo.lock
└── README.md
```

## Hacking

Install `just` to run project-specific commands.

```bash
# Install `cargo` via rustup if you don't already have it.
curl https://sh.rustup.rs -sSf | sh

# Install just
cargo install just
```

You can view the list of available development commands with `just -ls`.

Ref: [github.com/casey/just](https://github.com/casey/just)