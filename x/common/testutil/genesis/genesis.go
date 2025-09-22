package genesis

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_7_0"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// NewTestGenesisState returns [app.GenesisState] using the default genesis as input.
// The blockchain genesis state is represented as a map from module identifier
// strings to raw json messages.
func NewTestGenesisState(appCodec codec.Codec) app.GenesisState {
	genState := app.ModuleBasics.DefaultGenesis(appCodec)

	// Set short voting period to allow fast gov proposals in tests
	{
		var govGenState govtypes.GenesisState
		appCodec.MustUnmarshalJSON(genState[gov.ModuleName], &govGenState)
		*govGenState.Params.VotingPeriod = 20 * time.Second
		govGenState.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1e6)) // min deposit of 1 NIBI
		jsonBz, err := appCodec.MarshalJSON(&govGenState)
		if err != nil {
			panic(fmt.Errorf("failed to marshal gov genesis: %w", err))
		}
		genState[gov.ModuleName] = jsonBz
	}

	testapp.SetDefaultSudoGenesis(genState)

	// EVM Genesis - Set WNIBI to mimic mainnet (Nibiru v2.7.0)
	{
		var evmGenState evm.GenesisState
		moduleName := evm.ModuleName

		err := appCodec.UnmarshalJSON(genState[moduleName], &evmGenState)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal genesis from default: moduleName %s: %w", moduleName, err))
		}
		evmGenState.Accounts = append(evmGenState.Accounts, v2_7_0.WNIBI_GENESIS_EVM_ACC())

		jsonBz, err := appCodec.MarshalJSON(&evmGenState)
		if err != nil {
			panic(fmt.Errorf("failed to marshal evm genesis: %w", err))
		}
		genState[moduleName] = jsonBz
	}

	{
		var authGenState auth.GenesisState
		moduleName := auth.ModuleName

		err := appCodec.UnmarshalJSON(genState[moduleName], &authGenState)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal genesis from default: moduleName %s: %w", moduleName, err))
		}
		wnibiAcc := v2_7_0.WNIBI_GENESIS_AUTH_ACC()
		nextOpenAccNumber := len(authGenState.Accounts)
		wnibiAcc.AccountNumber = uint64(nextOpenAccNumber)

		// error not possible here because of compile-time asserts in
		// eth_account.go implementation of eth.EthAccount
		protobufAnyAccs, _ := auth.PackAccounts(auth.GenesisAccounts{
			&wnibiAcc,
		})
		wnibiAccAsProtobufAny := protobufAnyAccs[0]
		authGenState.Accounts = append(authGenState.Accounts, wnibiAccAsProtobufAny)

		jsonBz, err := appCodec.MarshalJSON(&authGenState)
		if err != nil {
			panic(fmt.Errorf("failed to marshal auth genesis: %w", err))
		}
		genState[moduleName] = jsonBz
	}

	return genState
}
