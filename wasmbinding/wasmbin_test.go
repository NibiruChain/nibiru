package wasmbinding_test

import (
	"fmt"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding/wasmbin"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

func init() {
	testapp.EnsureNibiruPrefix()
}

// TestSetupContracts acts as an integration test by storing and instantiating
// each production smart contract is expected to interact with x/wasm/binding.
func TestSetupContracts(t *testing.T) {
	sender := testutil.AccAddress()
	nibiru, _ := testapp.NewNibiruTestAppAndContext()
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})
	coins := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.NewInt(10)))
	require.NoError(t, testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))
	_, _ = SetupAllContracts(t, sender, nibiru, ctx)
}

var ContractMap = make(map[wasmbin.WasmKey]sdk.AccAddress)

// SetupAllContracts stores and instantiates all of wasm binding contracts.
func SetupAllContracts(
	t *testing.T, sender sdk.AccAddress, nibiru *app.NibiruApp, ctx sdk.Context,
) (*app.NibiruApp, sdk.Context) {
	wasmKey := wasmbin.WasmKeyPerpBinding
	codeId := StoreContract(t, wasmKey, ctx, nibiru, sender)
	deposit := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, sdk.OneInt()))
	contract := Instantiate.PerpBindingContract(t, ctx, nibiru, codeId, sender, deposit)
	ContractMap[wasmKey] = contract

	wasmKey = wasmbin.WasmKeyShifter
	codeId = StoreContract(t, wasmKey, ctx, nibiru, sender)
	contract = Instantiate.ShifterContract(t, ctx, nibiru, codeId, sender, deposit)
	ContractMap[wasmKey] = contract

	wasmKey = wasmbin.WasmKeyController
	codeId = StoreContract(t, wasmKey, ctx, nibiru, sender)
	contract = Instantiate.ControllerContract(t, ctx, nibiru, codeId, sender, deposit)
	ContractMap[wasmKey] = contract

	return nibiru, ctx
}

// StoreContract submits Wasm bytecode for storage on the chain.
func StoreContract(
	t *testing.T,
	wasmKey wasmbin.WasmKey,
	ctx sdk.Context,
	nibiru *app.NibiruApp,
	sender sdk.AccAddress,
) (codeId uint64) {
	pathToWasmBin := wasmbin.GetPackageDir(t) + "/wasmbin"
	wasmBytecode, err := wasmKey.ToByteCode(pathToWasmBin)
	require.NoError(t, err)

	// The "Create" fn is private on the nibiru.WasmKeeper. By placing it as the
	// decorated keeper in PermissionedKeeper type, we can access "Create" as a
	// public fn.
	wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
	instantiateAccess := &wasmtypes.AccessConfig{
		Permission: wasmtypes.AccessTypeEverybody,
	}
	codeId, _, err = wasmPermissionedKeeper.Create(
		ctx, sender, wasmBytecode, instantiateAccess,
	)
	require.NoError(t, err)
	return codeId
}

func InstantiateContract(
	t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, codeId uint64,
	initMsg []byte, sender sdk.AccAddress, label string, deposit sdk.Coins,
) (contractAddr sdk.AccAddress) {
	wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
	contractAddr, _, err := wasmPermissionedKeeper.Instantiate(
		ctx, codeId, sender, sender, initMsg, label, deposit,
	)
	require.NoError(t, err)
	return contractAddr
}

// Instantiate is a empty struct type with conventience functions for
// instantiating specific smart contracts.
var Instantiate = inst{}

type inst struct{}

func (i inst) PerpBindingContract(
	t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, codeId uint64,
	sender sdk.AccAddress, deposit sdk.Coins,
) (contractAddr sdk.AccAddress) {
	initMsg := []byte("{}")
	label := "x/perp module bindings"
	return InstantiateContract(
		t, ctx, nibiru, codeId, initMsg, sender, label, deposit,
	)
}

// Instantiates the shifter contract with the sender set as the admin.
func (i inst) ShifterContract(
	t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, codeId uint64,
	sender sdk.AccAddress, deposit sdk.Coins,
) (contractAddr sdk.AccAddress) {
	initMsg := []byte(fmt.Sprintf(`{ "admin": "%s"}`, sender))
	label := "contract for calling peg shift and depth shift in x/perp"
	return InstantiateContract(
		t, ctx, nibiru, codeId, initMsg, sender, label, deposit,
	)
}

// Instantiates the controller contract with the sender set as the admin.
func (i inst) ControllerContract(
	t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, codeId uint64,
	sender sdk.AccAddress, deposit sdk.Coins,
) (contractAddr sdk.AccAddress) {
	initMsg := []byte(fmt.Sprintf(`{ "admin": "%s"}`, sender))
	label := "contract for admin functions"
	return InstantiateContract(
		t, ctx, nibiru, codeId, initMsg, sender, label, deposit,
	)
}
