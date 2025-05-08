# x/evm/embeds/HACKING.md

- [Building Outputs](#building-outputs)
- [Precompile Solidity Documentation](#precompile-solidity-documentation)
  - [Comments](#comments)
  - [NatSpec Fields](#natspec-fields)
- [Solidity Conventions](#solidity-conventions)

## Building Outputs 

Workhorse command
```bash
just gen-embeds
```

From inside the "Nibiru/x/evm/embeds" directory
```bash
yarn --check-files
yarn hardhat compile && echo "SUCCESS: yarn hardhat compile succeeded" || echo "Run failed"
```

## Publishing

After incrementing the version number in the `package.json`:

```bash
just gen-embeds # if you've not already
npm publish --access public
```

This suggests a hyperlink to authorize yourself on `npm` and then publish a
new version of the [`@nibiruchain/solidity` package](https://nibiru.fi/docs/dev/evm/npm-solidity.html).

## Solidity Documentation for the Nibiru Precompiles

Example of a well-documented contract: [[Uniswap/v4-core/.../IHooks.sol](https://github.com/Uniswap/v4-core/blob/3407bce4b39869fe41ad5ec724b2df308c34900f/src/interfaces/IHooks.sol)]

### Comments

You should use `///` for Solidity comments to document code in the NatSpec
(Ethereum Natural Specification) format. Many tools like Solidity IDEs, plugins,
and documentation generators use NatSpec comments.

### NatSpec Fields

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
/// @notice The hook called before liquidity is removed
/// @param sender The initial msg.sender for the remove liquidity call
/// @param key The key for the pool
/// @param params The parameters for removing liquidity
/// @param hookData Arbitrary data handed into the PoolManager by the liquidity provider to be passed on to the hook
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

## Solidity Conventions

### State Mutability

State mutability defines how a function interacts with the blockchain state. Always explicitly declare non-default state mutability keywords for clarity and correctness.

1. `view` : For stateful queries
   - Reads state, but cannot modify it.  
   - Use for getters or queries.  

   ```solidity
   function getBalance(address account) external view returns (uint256);
   ```

2. `pure` : For stateless queries
   - Neither reads nor modifies state.  
   - Use for calculations or logic relying only on inputs.  

   ```solidity
   function add(uint256 a, uint256 b) external pure returns (uint256);
   ```

3. `nonpayable` : (Default) State mutating operation
   - Modifies state but cannot receive Ether.  
   - Default if no mutability is specified.  

   ```solidity
   function updateBalance(address account, uint256 amount) external;
   ```

4. `payable` : State mutating operation that can receive Ether (NIBI)
   - Can receive Ether and may modify state.  
   - Use for deposits or payments.  

   ```solidity
   function deposit() external payable;
   ```
