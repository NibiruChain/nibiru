---
order: 5
---
# 🖼️ NFTs

NFTs, valued for their uniqueness, are digital assets commonly used
in digital art, collectibles, and gaming. To deploy an NFT contract
on Nibiru Chain, developers can customize the CW721-based contract
found in the cw-nfts repository to adhere to the CW721 standard,
then use the nibid CLI for deployment.{synopsis}

## Non-Fungible Tokens (NFTs)

Non-Fungible Tokens (NFTs) are a type of digital asset that is unique
and indivisible, each possessing distinct identities and properties.
Unlike fungible tokens, which are interchangeable on a one-to-one basis,
NFTs cannot be exchanged in a like-for-like manner due to their unique
attributes and characteristics. These unique attributes are typically
encoded in the token's metadata, which includes information such as the
token's creator, creation date, and any additional properties that make
it distinct from other tokens in the same series or collection.

In the blockchain industry, NFTs have gained significant traction and are
used in various applications such as digital art, collectibles, and gaming.
NFTs leverage blockchain technology to provide irrefutable proof of ownership
and provenance, ensuring that each token is verifiably scarce and authentic.

## Setting Up Your NFT Contract

On Nibiru Chain, developers can deploy NFT contracts following the `CW721` standard.
This standard defines a set of interfaces and methods that enable the creation,
transfer, and management of NFTs on the blockchain. Developers can customize the
`CW721-based` contract from the cw-nfts repository to suit their specific requirements
and deploy it using the nibid CLI. Once deployed, these NFTs can be bought, sold, and
traded on the Nibiru Chain, leveraging the security and transparency of blockchain technology.

In order to create your very own NFT contract on Nibiru, you must first set your
environemnt bu following the [Getting Started guide](./getting-started.md).

Next, clone the `CW721-base` contract from the `cw-nfts` [repository](https://github.com/CosmWasm/cw-nfts/blob/main/packages/cw721/README.md).

```bash
git clone https://github.com/CosmWasm/cw-nfts.git
cd cw-nfts/contracts/cw721-base
```

To verify and test your setup, run to ensure all pre-exting tests succeed:

```bash
cargo test
```

## Understanding & Customizing Cw721-base

The `CW721-base` contract serves as a foundational template for creating non-fungible tokens (NFTs) on the CosmWasm blockchain. It adheres to the CW721 standard and provides functionality for minting new tokens, transferring ownership, and querying token information. Developers can customize this contract by extending its functionality or incorporating its logic into other contracts to create CW721-compatible NFTs with bespoke features.

To customize the `CW721-base` contract, it is essential to understand its implementation and structure. The contract's logic can be found [here](https://github.com/CosmWasm/cw-nfts/tree/main/contracts/cw721-base). Familiarize yourself with this codebase to modify it according to your project's specifications and requirements. This understanding will enable you to tailor the contract to suit your unique use cases, ensuring compatibility with the CW721 standard while incorporating your desired functionality.

## Build the Contract

The process of deploying the contract mirrors that of any other CW contract. Initially, the contract is compiled into a binary `.wasm` file. Subsequently, optimization is carried out using the [CosmWasm Rust Optimizer](https://github.com/CosmWasm/optimizer).

```bash
cargo wasm
```

```bash
# return to cw-nfts/ directory
cd ../..

docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.15.0 ./contracts/cw721-base/
```

This will generate the optimized `.wasm` binary contract in `artifacts/cw721-base.wasm`.

## Deploy & Instantiate the Contract

Interacting with the contract similar to how you manage any [CosmWasm contract](./cw-manage.md):

```bash
ACCOUNT=<your-wallet-address>

TXHASH="$(nibid tx wasm store artifacts/cw721_base.wasm \
    --from $ACCOUNT \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq -rcs '.[0].txhash')"
```

```bash
# Query the transaction hash:
nibid q tx $TXHASH > txhash.json

# Save the CODE_ID for later usage:
CODE_ID="$(cat txhash.json | jq -r '.logs[0].events[1].attributes[1].value')"
```

Instantiate your contract using the `$CODE_ID`, but first let's save the instantiation message:

```bash
COLLECTION_NAME=<your-collection-name>
SYMBOL=<your symbol>

echo "{\"name\":\"$COLLECTION_NAME\", \"symbol\":\"$SYMBOL\"}" | jq . | tee instantiate_args.json
```

```bash
TXHASH_INIT="$(nibid tx wasm instantiate $CODE_ID \
    "$(cat instantiate_args.json)" \
    --admin "$ACCOUNT" \
    --label CALICO_NFTS \
    --from $ACCOUNT \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.025unibi \
    --yes | jq -rcs '.[0].txhash')"
```

```bash
# Query the transaction
nibid q tx $TXHASH_INIT > txhash.init.json

# Save the contract address
CONTRACT_ADDRESS="$(cat txhash.init.json | jq -r '.logs[0].events[1].attributes[0].value')"
```

With this, you have your very own NFT contract address which you can utilize futher.

## Related Pages

- [Getting Started](./getting-started.md)
- [Managing Smart Contracts](./cw-manage.md)
