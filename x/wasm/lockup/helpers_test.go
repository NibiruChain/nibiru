package lockup_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/common/testutil/mock"

	"github.com/NibiruChain/nibiru/app"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/wasm/lockup"
)

func SetupCustomApp(t *testing.T, addr sdk.AccAddress) (*app.NibiruApp, sdk.Context) {
	nibiru, ctx := CreateTestInput()
	wasmKeeper := nibiru.WasmKeeper

	storeLockupCode(t, ctx, nibiru, addr)

	cInfo := wasmKeeper.GetCodeInfo(ctx, 1)
	require.NotNil(t, cInfo)

	return nibiru, ctx
}

func JoinPool(t *testing.T, app *app.NibiruApp, ctx sdk.Context) (sdk.AccAddress, sdk.Coin) {
	const shareDenom = "nibiru/pool/1"

	// Test values
	poolAddr := testutil.AccAddress()
	initialPool := mock.SpotPool(
		/*poolId=*/ 1,
		/*assets=*/ sdk.NewCoins(
			sdk.NewInt64Coin("bar", 100),
			sdk.NewInt64Coin("foo", 100),
		),
		/*shares=*/ 100,
	)
	joinerInitialFunds := sdk.NewCoins(
		sdk.NewInt64Coin("bar", 100),
		sdk.NewInt64Coin("foo", 100),
	)

	// Expected values
	expectedFinalPool := mock.SpotPool(
		/*poolId=*/ 1,
		/*assets=*/ sdk.NewCoins(
			sdk.NewInt64Coin("bar", 200),
			sdk.NewInt64Coin("foo", 200),
		),
		/*shares=*/ 200,
	)
	expectedNumSharesOut := sdk.NewInt64Coin(shareDenom, 100)
	expectedRemCoins := sdk.NewCoins()

	// Create a new pool and add some liquidity
	initialPool.Address = poolAddr.String()
	expectedFinalPool.Address = poolAddr.String()
	app.SpotKeeper.SetPool(ctx, initialPool)

	joinerAddr := testutil.AccAddress()
	require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, joinerInitialFunds))

	pool, numSharesOut, remCoins, err := app.SpotKeeper.JoinPool(ctx, joinerAddr, 1, joinerInitialFunds, false)
	require.NoError(t, err)
	require.Equal(t, expectedFinalPool, pool)
	require.Equal(t, expectedNumSharesOut, numSharesOut)
	require.Equal(t, expectedRemCoins, remCoins)

	return joinerAddr, expectedNumSharesOut
}

func storeLockupCode(t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, addr sdk.AccAddress) {
	wasmCode, err := os.ReadFile("./testdata/lockup.wasm")
	require.NoError(t, err)

	contractKeeper := keeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
	_, _, err = contractKeeper.Create(ctx, addr, wasmCode, &wasmtypes.AccessConfig{
		Permission: wasmtypes.AccessTypeEverybody,
	})
	require.NoError(t, err)
}

func instantiateLockupContract(t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, funder sdk.AccAddress) sdk.AccAddress {
	initMsgBz := []byte("{}")
	contractKeeper := keeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
	codeID := uint64(1)
	addr, _, err := contractKeeper.Instantiate(ctx, codeID, funder, funder, initMsgBz, "lockup contract", nil)
	require.NoError(t, err)

	return addr
}

func executeCustom(t *testing.T, ctx sdk.Context, app *app.NibiruApp, contract sdk.AccAddress, sender sdk.AccAddress, msg lockup.LockupMsg, funds sdk.Coin) error {
	perpBz, err := json.Marshal(msg)
	require.NoError(t, err)

	// no funds sent if amount is 0
	var coins sdk.Coins
	if !funds.Amount.IsNil() {
		coins = sdk.Coins{funds}
	}

	contractKeeper := keeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	_, err = contractKeeper.Execute(ctx, contract, sender, perpBz, coins)
	return err
}

func queryCustom(t *testing.T, ctx sdk.Context, app *app.NibiruApp, contract sdk.AccAddress, request lockup.LockupQuery, response interface{}, shouldFail bool) {
	queryBz, err := json.Marshal(request)
	require.NoError(t, err)

	resBz, err := app.WasmKeeper.QuerySmart(ctx, contract, queryBz)
	if shouldFail {
		require.Error(t, err)
		return
	}

	require.NoError(t, err)
	err = json.Unmarshal(resBz, response)
	require.NoError(t, err)
}

func CreateTestInput() (*app.NibiruApp, sdk.Context) {
	encoding := app.MakeTestEncodingConfig()
	app := testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encoding.Marshaler))
	ctx := app.BaseApp.NewContext(false, tmproto.Header{Height: 1, ChainID: "kujira-1", Time: time.Now().UTC()})
	return app, ctx
}

func FundAccount(t *testing.T, ctx sdk.Context, app *app.NibiruApp, acct sdk.AccAddress) {
	err := simapp.FundAccount(app.BankKeeper, ctx, acct, sdk.NewCoins(
		sdk.NewCoin("uosmo", sdk.NewInt(10000000000)),
	))
	require.NoError(t, err)
}

// we need to make this deterministic (same every test run), as content might affect gas costs
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

func RandomAccountAddress() sdk.AccAddress {
	_, _, addr := keyPubAddr()
	return addr
}

func RandomBech32AccountAddress() string {
	return RandomAccountAddress().String()
}
