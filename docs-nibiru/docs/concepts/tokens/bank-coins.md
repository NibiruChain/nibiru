---
order: 3
metaTitle: "Bank Coins | Fungible Token Standards on Nibiru"
---

# Bank Coins

The Bank Coin is one of the primary fungible token standards used on the Nibiru
blockchain alongside ERC20s. This standardization ensures that wallets,
exchanges, and smart contracts can interact with all Bank Coins in a consistent
manner, similar to how they handle the native NIBI token.

## Fungible Assets

In the Nibiru ecosystem, the `sdk.Coin` type represents fungible assets. Each
Bank Coin is identified by a unique denomination (denom) and contains a specific
amount. The denom serves as a unique identifier for the asset type, while the
amount represents the quantity of that asset.

## Fungibility: Bank Coins versus NFTs

Fungible tokens are equivalent and interchangeable digital assets. These includes
familiar tokens like NIBI, ETH, fiat currencies like KRW and USD, and
quantifiable voting rights. Non-fungible tokens are distinct and unique and uses
to represent items like collectibles, licenses, or deeds of ownership.

## Bank Coins as Fungible Tokens 

The "Bank Coin" is one the main standards for fungible tokens on Nibiru alongside ERC20 tokens. Bank Coins are quite simple and described by two pieces of information: a denomination and an amount. 

```go
// JSON example: {"denom": "btc", "amount": "123456"}
type BankCoin struct {
    Denom  string   `json:"denom"`
    Amount *big.Int `json:"amount"`
}
```

A coin denomination, or bank `denom` for short, is a unique identifier for the asset to be understood by the Nibiru blockchain. For example, NIBI tokens have bank denom "unibi".

**Smart Contract Support**: All Wasm smart contracts on Nibiru can hold or receive coins. Special contract logic is not needed to enable protocol.

**Wallet Compatibility**: Bank Coins are compatible with a wide range of wallets and decentralized applications within the IBC ecosystem.

**Multi-Chain Compatibility**: Bank Coins are a multi-chain standard. They have bridge and IBC support by default. Coins can only be created by the Bank Module on Nibiru, which is essentially an ownerless contract that can mint or burn tokens and manages all transfers of coins.

## How do Bank Coins differ from ERC20 tokens?

| Aspect | Bank Coins | ERC-20 Tokens |
|--------|------------|----------------|
| State Management | Managed directly by the Bank module in its own key-value store | Managed within each token's smart contract state |
| Creation Process | Created through governance proposals, permissioned modules, or the Tokenfactory module | Created by deploying new smart contracts |
| Efficiency | Typically more gas-efficient due to native implementation | Generally consume more gas due to smart contract execution |
| Standardization | All follow the same standard and are handled uniformly by the blockchain | Can have variations in implementation, though most follow a standard interface |
| Balance Queries | Queried through the Bank module's API, which is easy to manage because the same generic query is used for all coins. | Queried by calling the ERC20 token's smart contract, as balances are individually stored on each contract. |
| Transfer Mechanism | Executed directly by the Bank module | Executed by calling the token's transfer function |
| Supply Management | Controlled by authorized modules or governance | Managed by the token's smart contract logic |
| Permissionless Creation | Possible via the Tokenfactory module | Inherent (anyone can deploy a new token contract) |
| Storage Efficiency | More efficient as all coins share the same infrastructure | Each token contract stores its own data separately |
| Interoperability | Native to the Bank Module but can be converted to ERC20 via the FunToken system | Native to EVM environments (Nibiru EVM in this case) but can be converted to Bank Coins via the FunToken system |

## Creating Bank Coins

Unlike custom token standards that require smart contract deployment, Bank Coins
are a native feature of Nibiru. They can be created through governance proposals
or by modules with the appropriate permissions.

### Coin Metadata

When creating a new Bank Coin, you can specify metadata that describes its properties:

- `Name`: The full name of the coin (e.g., "Nibiru")
- `Symbol`: A short identifier for the coin (e.g., "NIBI")
- `Description`: A brief explanation of the coin's purpose or characteristics
- `DenomUnits`: Defines the coin's denomination units and their conversion rates
- `Base`: The fundamental unit of the coin, used for internal calculations
- `Display`: The default unit for displaying the coin's value to users


## Transferring Bank Coins

Bank Coins can be freely transferred between accounts using the standard transfer functions provided by the bank module. These transfers are subject to the sender having sufficient balance and any additional logic implemented by custom modules.

## Interoperability with Smart Contracts

While Bank Coins are native to Nibiru, they can be seamlessly used within the EVM environment through the FunToken system. This allows for Bank Coins to be represented as ERC-20 tokens, enabling their use in smart contracts and DeFi applications.

## Key Differences from Custom Tokens

- Native Support: Bank Coins are natively supported by the Nibiru blockchain, requiring no additional smart contract logic for basic functionality.
- Standardized Handling: All modules and applications on Nibiru can interact with Bank Coins using a consistent API.
- Efficient: Operations with Bank Coins are generally more gas-efficient than custom token implementations.
- Governance Controlled: The creation and management of Bank Coins can be subject to on-chain governance decisions.

For more information on working with Bank Coins in your applications or modules, refer to the following resources:

- [Creating a New Bank Coin](link-to-guide): A guide on proposing and implementing a new Bank Coin.
- [Bank Module Documentation](link-to-docs): Detailed documentation on all available functions for managing Bank Coins.
- [Nibiru Tokenomics](link-to-tokenomics): Learn about the role of Bank Coins in the broader Nibiru ecosystem.

## References

- [GitHub - Nibiru Rust (nibiru-wasm)](https://github.com/NibiruChain/nibiru-wasm)
- [GitHub - Nibiru Blockchain (nibiru)](https://github.com/NibiruChain/nibiru)
- [Token Factory Module | Nibiru](../../arch/tokenfactory.md)
- [Bank Module | Nibiru](../../arch/bank.md)
