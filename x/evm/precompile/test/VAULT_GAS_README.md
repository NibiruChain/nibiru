# Vault Gas Measurement Tests

This directory contains contracts and tests for measuring gas consumption when interacting with WASM vault contracts through EVM.

## Files

- `vault.wasm` - The vault contract (CosmWasm)
- `vault_token_minter.wasm` - Token minter contract for vault shares
- `SimpleVaultInterface.sol` - Simplified Solidity interface for vault deposits
- `../vault_gas_test.go` - Go test suite for measuring gas consumption

## Test Setup

1. **Deploy WASM Contracts**: The test automatically deploys the vault and minter contracts
2. **Create FunToken**: Sets up ERC20 representation of the collateral token
3. **Measure Gas**: Tests various operations and measures gas consumption

## Gas Measurement Points

The tests measure gas consumption at different stages:

1. **Direct WASM Operations**
   - Direct deposit to vault (baseline)
   - Direct query operations

2. **Precompile Operations**
   - WASM operations through EVM precompile
   - Overhead measurement vs direct calls

3. **Token Conversions**
   - ERC20 to Bank token conversion (sendToBank)
   - Bank to ERC20 conversion (sendToEvm)

4. **Complete Flow**
   - Full deposit flow: ERC20 → Bank → Vault
   - Measures total gas for user-facing operations

## Running Tests

```bash
# Run all vault gas tests
go test -v -run TestVaultGasConsumption ./x/evm/precompile/...

# Run specific test
go test -v -run TestVaultGasConsumption/TestDirectWasmVaultDeposit ./x/evm/precompile/...
```

## Key Findings

The tests help identify:
- Gas multiplication factor between WASM and EVM
- Overhead from precompile infrastructure  
- Cost of token conversions
- Total cost for complete user operations

## Contract Instantiation

The vault requires specific initialization parameters:
- `collateral_denom`: The bank denomination for collateral (e.g., "utest")
- `vault_token_minter_contract`: Address of the minter contract
- Other parameters can be dummy values for deposit testing

The minter contract requires:
- `denom`: Share token denomination (e.g., "vault_shares")
- `owner`: Address that can mint shares