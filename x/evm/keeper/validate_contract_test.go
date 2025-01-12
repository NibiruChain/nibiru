package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

// TestHasMethodInContract_RealKeeper deploys a real ERC20 contract and tests
// the presence/absence of a couple of methods using the actual keeper logic.
func TestHasMethodInContract_RealKeeper(t *testing.T) {
	// 1) Build standard test dependencies
	deps := evmtest.NewTestDeps()
	ctx := sdk.WrapSDKContext(deps.Ctx)
	k := deps.App.EvmKeeper

	// 2) Deploy the standard ERC20 (Minter) contract
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20Minter,
		"ExampleToken",
		"EXM",
		uint8(18),
	)
	require.NoError(t, err, "error deploying ERC20 test contract")

	// 3) The embedded ERC20 ABI includes balanceOf, transfer, decimals, etc.
	erc20Abi := embeds.SmartContract_ERC20Minter.ABI

	// For demonstration, let's see if the contract implements "balanceOf"
	methodBalanceOf, ok := erc20Abi.Methods["balanceOf"]
	require.True(t, ok, `"balanceOf" not found in the ERC20 ABI?`)

	// Now let's see if the keeper says "balanceOf" is recognized
	hasMethod, err := k.HasMethodInContract(ctx, deployResp.ContractAddr, methodBalanceOf)
	require.NoError(t, err)
	require.True(t, hasMethod, "expected contract to have 'balanceOf'")

	// 4) Next, let's test a fake method that doesn't exist
	fakeMethod := methodBalanceOf
	fakeMethod.Name = "someFakeMethod"
	fakeMethod.ID = []byte{0xde, 0xad, 0xbe, 0xef} // random 4-byte selector

	hasMethod, err = k.HasMethodInContract(ctx, deployResp.ContractAddr, fakeMethod)
	require.NoError(t, err, "non-existent method calls shouldn't produce a real EVM error")
	require.False(t, hasMethod, "expected the contract to NOT have 'someFakeMethod'")
}

// TestCheckAllMethods_RealKeeper uses your keeper’s checkAllethods (assuming
// you renamed it from “checkAllMethods” to a public name).
func TestCheckAllMethods_RealKeeper(t *testing.T) {
	// Build test dependencies and context
	deps := evmtest.NewTestDeps()
	ctx := sdk.WrapSDKContext(deps.Ctx)
	k := deps.App.EvmKeeper

	// Deploy a standard ERC20 contract
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20Minter,
		"DemoToken",
		"DMO",
		uint8(6),
	)
	require.NoError(t, err)

	// Example: We want to check that it has "balanceOf" and "transfer", but *not* "fakeMethod"
	erc20Abi := embeds.SmartContract_ERC20Minter.ABI

	// Gather the actual method objects from the ABI
	balanceOfMethod, hasBalanceOf := erc20Abi.Methods["balanceOf"]
	require.True(t, hasBalanceOf)
	transferMethod, hasTransfer := erc20Abi.Methods["transfer"]
	require.True(t, hasTransfer)

	// Let's also define a known-fake method
	fakeMethod := abi.Method{
		Name: "fakeMethod",
		ID:   []byte{0xfa, 0x75, 0x55, 0x0f}, // random
	}

	// Scenario 1: "balanceOf" + "transfer" => no error
	allMethods := []abi.Method{balanceOfMethod, transferMethod}
	err = k.CheckAllethods(ctx, deployResp.ContractAddr, allMethods)
	require.NoError(t, err, "both balanceOf and transfer exist in standard ERC20")

	// Scenario 2: "balanceOf" + "fakeMethod" => we expect an error on second
	calls := []abi.Method{balanceOfMethod, fakeMethod}
	err = k.CheckAllethods(ctx, deployResp.ContractAddr, calls)
	require.Error(t, err, "contract does not have 'fakeMethod'")
	require.Contains(t, err.Error(), "fakeMethod not found in contract")
}
