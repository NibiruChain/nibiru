package keeper_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	codec "github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/tokenfactory/fixture"
	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// Instantiate is a empty struct type with conventience functions for
// instantiating specific smart contracts.
var Instantiate = inst{}

type inst struct{}

func (i inst) ContractNibiStargate(
	t *testing.T, ctx sdk.Context, nibiru *app.NibiruApp, codeId uint64,
	sender sdk.AccAddress, deposit sdk.Coins,
) (contractAddr sdk.AccAddress) {
	initMsg := []byte("{}")
	label := "token factory stargate message examples"
	return InstantiateContract(
		t, ctx, nibiru, codeId, initMsg, sender, label, deposit,
	)
}

func (s *TestSuite) ExecuteAgainstContract(
	contract LiveContract, execMsgJson string,
) (contractRespBz []byte, err error) {
	execMsg := json.RawMessage([]byte(execMsgJson))
	return wasmkeeper.NewDefaultPermissionKeeper(s.app.WasmKeeper).Execute(
		s.ctx, contract.Addr, contract.Deployer, execMsg, sdk.Coins{},
	)
}

type LiveContract struct {
	CodeId   uint64
	Addr     sdk.AccAddress
	Deployer sdk.AccAddress
}

var LiveContracts = make(map[string]LiveContract)

// SetupContracts stores and instantiates all of the CosmWasm smart contracts.
func SetupContracts(
	t *testing.T, sender sdk.AccAddress, nibiru *app.NibiruApp, ctx sdk.Context,
) map[string]LiveContract {
	wasmName := fixture.WASM_NIBI_STARGATE
	codeId := StoreContract(t, wasmName, ctx, nibiru, sender)
	deposit := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, math.OneInt()))
	contract := Instantiate.ContractNibiStargate(t, ctx, nibiru, codeId, sender, deposit)
	LiveContracts[wasmName] = LiveContract{
		CodeId:   codeId,
		Addr:     contract,
		Deployer: sender,
	}

	return LiveContracts
}

// StoreContract submits Wasm bytecode for storage on the chain.
func StoreContract(
	t *testing.T,
	wasmName string,
	ctx sdk.Context,
	nibiru *app.NibiruApp,
	sender sdk.AccAddress,
) (codeId uint64) {
	// Read wasm bytecode from disk
	pkgDir, err := testutil.GetPackageDir()
	require.NoError(t, err)
	pathToModulePkg := path.Dir(pkgDir)
	require.Equal(t, tftypes.ModuleName, path.Base(pathToModulePkg))
	pathToWasmBin := pathToModulePkg + fmt.Sprintf("/fixture/%s", wasmName)
	wasmBytecode, err := os.ReadFile(pathToWasmBin)
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

type WasmTestCase struct {
	ExecMsgJson string
	WantErr     string
	TestCaseTx
}

// TestStargate: Showcases happy path examples for tokenfactory messages
// executed as `CosmosMsg::Stargate` types built directly from protobuf.
//
// in the example smart contract.
func (s *TestSuite) TestStargate() {
	s.T().Log("create contract deployer and fund account")
	deployer, err := sdk.AccAddressFromBech32("nibi18wcr5svu0dexdj2zwk44hcjfw6drcsfkn6hq9q")
	s.NoError(err)
	funds, err := sdk.ParseCoinsNormalized("69000000unibi") // just for gas
	s.NoError(err)
	s.NoError(
		testapp.FundAccount(s.app.BankKeeper, s.ctx, deployer, funds),
	)

	liveContracts := SetupContracts(s.T(), deployer, s.app, s.ctx)
	contract, isFound := liveContracts[fixture.WASM_NIBI_STARGATE]
	s.True(isFound)

	tfdenom := tftypes.TFDenom{
		Creator:  contract.Addr.String(),
		Subdenom: "zzz",
	}
	s.Run("create denom from smart contract", func() {
		_, err := s.ExecuteAgainstContract(contract, strings.Trim(`
		{
			"create_denom": { "subdenom": "zzz" }
		}
		`, " "))
		s.NoError(err)

		// NOTE that the smart contract is the sender.
		denoms := s.app.TokenFactoryKeeper.QueryDenoms(s.ctx,
			contract.Addr.String(),
		)
		s.ElementsMatch(denoms, []string{tfdenom.Denom().String()})
	})

	someoneElse := testutil.AccAddress()
	s.Run("mint from smart contract", func() {
		execMsgJson := strings.Trim(fmt.Sprintf(`
		{ 
			"mint": { 
				"coin": { "amount": "69420", "denom": "%s" }, 
				"mint_to": "%s" 
			} 
		}
		`, tfdenom.Denom(), someoneElse), " ")
		_, err := s.ExecuteAgainstContract(contract, execMsgJson)
		s.NoError(err, "execMsgJson: %v", execMsgJson)

		balance := s.app.BankKeeper.GetBalance(s.ctx, someoneElse, tfdenom.Denom().String())
		s.Equal(math.NewInt(69_420), balance.Amount)
	})

	s.Run("burn from smart contract", func() {
		execMsgJson := strings.Trim(fmt.Sprintf(`
		{ 
			"burn": { 
				"coin": { "amount": "69000", "denom": "%s" }, 
				"burn_from": "%s" 
			} 
		}
		`, tfdenom.Denom(), someoneElse), " ")
		_, err := s.ExecuteAgainstContract(contract, execMsgJson)
		s.NoError(err, "execMsgJson: %v", execMsgJson)

		balance := s.app.BankKeeper.GetBalance(s.ctx, someoneElse, tfdenom.Denom().String())
		s.Equal(math.NewInt(420), balance.Amount)
	})

	s.Run("change admin from smart contract", func() {
		execMsgJson := strings.Trim(fmt.Sprintf(`
		{ 
			"change_admin": { 
				"denom": "%s", 
				"new_admin": "%s" 
			} 
		}
		`, tfdenom.Denom(), someoneElse), " ")
		_, err := s.ExecuteAgainstContract(contract, execMsgJson)
		s.NoError(err, "execMsgJson: %v", execMsgJson)

		denomInfo, err := s.app.TokenFactoryKeeper.QueryDenomInfo(
			s.ctx, tfdenom.Denom().String(),
		)
		s.NoError(err)
		s.Equal(someoneElse.String(), denomInfo.Admin)
		s.Equal(tfdenom.DefaultBankMetadata(), denomInfo.Metadata)
	})
}

/*
TestStargateSerde: Compares marshaled bytes produced for an sdk.Msg against
the equivalent encoding in Rust (`Vec<u8>`).
Test values for the `wantBz` field can be reproduced for a
`nibiru_std::proto::NibiruStargateMsg` by running the `to_bytes` function.

Example:

	```rust
	use nibiru_std::proto::nibiru;
	let pb_msg: NibiruProstMsg = nibiru::tokenfactory::MsgMint { ... }
	println!("{:?}", pb_msg.to_bytes())
	```

The printed vector corresponds one-to-one with what we'd get when marshaling
that protobuf message to `[]byte` in Go.
*/
func (s *TestSuite) TestStargateSerde() {
	testCases := []struct {
		// sdkMsg: A protobuf message implementing both the sdk.Msg and
		// codec.ProtoMarshaler interfaces. Any transaction message will work here.
		// Note that you need the pointer to the message for encoding.
		sdkMsg any
		// A namespaced string identifier for the type of a serialized protobuf
		// message. Often associated with the `Any` type.
		typeUrl string
		wantBz  string
	}{
		{
			typeUrl: "/nibiru.tokenfactory.v1.MsgCreateDenom",
			sdkMsg: &tftypes.MsgCreateDenom{
				Sender:   "sender",
				Subdenom: "subdenom",
			},
			wantBz: "[10 6 115 101 110 100 101 114 18 8 115 117 98 100 101 110 111 109]",
		},
		{
			typeUrl: "/nibiru.tokenfactory.v1.MsgMint",
			sdkMsg: &tftypes.MsgMint{
				Sender: "sender",
				Coin:   sdk.NewInt64Coin("abcxyz", 123),
				MintTo: "mint_to",
			},
			wantBz: fmt.Sprint([]byte{10, 6, 115, 101, 110, 100, 101, 114, 18, 13, 10, 6, 97, 98, 99, 120, 121, 122, 18, 3, 49, 50, 51, 26, 7, 109, 105, 110, 116, 95, 116, 111}),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.typeUrl, func() {
			pbMsg, _ := (tc.sdkMsg).(codec.ProtoMarshaler)
			sgMsgValue := s.encConfig.Marshaler.MustMarshal(pbMsg)
			sgMsg := wasmvmtypes.StargateMsg{
				TypeURL: tc.typeUrl,
				Value:   sgMsgValue,
			}
			if tc.wantBz != "" {
				bz, _ := parseByteList(tc.wantBz)
				s.Equal(bz, sgMsgValue)
			}

			ibcTransferPort := wasmtesting.MockIBCTransferKeeper{
				GetPortFn: func(ctx sdk.Context) string { return "myTransferPort" },
			}
			wasmEncoders := wasmkeeper.DefaultEncoders(s.encConfig.Marshaler, ibcTransferPort)
			mockContractAddr := testutil.AccAddress()
			sdkMsgs, err := wasmEncoders.Encode(s.ctx, mockContractAddr, "mock-ibc-port",
				wasmvmtypes.CosmosMsg{
					Stargate: &sgMsg,
				},
			)

			s.Require().NoError(err)
			sdkMsg, _ := (tc.sdkMsg).(sdk.Msg)
			s.EqualValues(sdkMsg, sdkMsgs[0])
		})
	}
}

// parseByteList: Parses the string version of a `[]byte` object when formatted
// using methods like 'fmt.Sprintf' or 'fmt.Println' to recover the underlying
// type again.
func parseByteList(bzStr string) ([]byte, error) {
	bzStr = strings.TrimPrefix(bzStr, "[")
	bzStr = strings.TrimSuffix(bzStr, "]")
	parts := strings.Split(bzStr, " ")

	var result []byte
	for _, part := range parts {
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		result = append(result, byte(val))
	}

	return result, nil
}
