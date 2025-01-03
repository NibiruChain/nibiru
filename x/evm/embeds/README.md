# Nibiru Contract Embeds

## Hacking

```shell
npm install
npx hardhat compile
```

## Precompile Solidity Documentation

Example of a well-documented contract: [[Uniswap/v4-core/.../IHooks.sol](https://github.com/Uniswap/v4-core/blob/3407bce4b39869fe41ad5ec724b2df308c34900f/src/interfaces/IHooks.sol)]

- `@notice`: Used to explain to end users what the function does. Should be written in plain English and focus on the function's purpose.  
  Best practice: Include for all public and external functions.
- `@param`: Describes a function parameter. Should explain what the parameter is used for.  
  Best practice: Include for all function parameters, especially in interfaces.
- `@dev`: Provides additional details for developers. Used for implementation details, notes, or warnings for developers.  
  Best practice: Use when there's important information that doesn't fit in `@notice` but is crucial for developers.
- `@return`: Describes what a function returns.  
  Best practice: Use for all functions that return values, explaining each return value.

Example from IHooks.sol:
```solidity
/@notice The hook called before liquidity is removed
/// @param sender The initial msg.sender for the remove liquidity call
/// @param key The key for the pool
/// @param params The parameters for removing liquidity
/// @param hookData Arbitrary data handed into the PoolManager by the liquidity provider to be be passed on to the hook
/// @return bytes4 The function selector for the hook
function beforeRemoveLiquidity(
    address sender,
    PoolKey calldata key,
    IPoolManager.ModifyLiquidityParams calldata params,
    bytes calldata hookData
) external returns (bytes4);
```

@inheritdoc:

Used to inherit documentation from a parent contract or interface.
Best practice: Use when you want to reuse documentation from a base contract.


@title:

Provides a title for the contract or interface.
Best practice: Include at the top of each contract or interface file.


@author:

States the author of the contract.
Best practice: Optional, but can be useful in larger projects.
