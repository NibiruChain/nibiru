package keepers

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"

	// ---------------------------------------------------------------
	// IBC imports

	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"

	// ---------------------------------------------------------------
	// Nibiru Custom Modules

	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
	epochskeeper "github.com/NibiruChain/nibiru/v2/x/epochs/keeper"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	inflationkeeper "github.com/NibiruChain/nibiru/v2/x/inflation/keeper"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
	sudokeeper "github.com/NibiruChain/nibiru/v2/x/sudo/keeper"
	tokenfactorykeeper "github.com/NibiruChain/nibiru/v2/x/tokenfactory/keeper"
)

type PublicKeepers struct {
	// AccountKeeper encodes/decodes accounts using the go-amino (binary) encoding/decoding library
	AccountKeeper authkeeper.AccountKeeper
	// BankKeeper defines a module interface that facilitates the transfer of coins between accounts
	BankKeeper    *evmkeeper.NibiruBankKeeper
	StakingKeeper *stakingkeeper.Keeper
	/* DistrKeeper is the keeper of the distribution store */
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper

	// make IBC modules public for test purposes
	// these modules are never directly routed to by the IBC Router
	FeeMockModule ibcmock.IBCModule

	// ---------------
	// Nibiru keepers
	// ---------------
	EpochsKeeper       epochskeeper.Keeper
	OracleKeeper       oraclekeeper.Keeper
	InflationKeeper    inflationkeeper.Keeper
	SudoKeeper         sudokeeper.Keeper
	DevGasKeeper       devgaskeeper.Keeper
	TokenFactoryKeeper tokenfactorykeeper.Keeper
	EvmKeeper          *evmkeeper.Keeper

	// WASM keepers
	WasmKeeper         wasmkeeper.Keeper
	WasmMsgHandlerArgs wasmext.MsgHandlerArgs

	ScopedWasmKeeper capabilitykeeper.ScopedKeeper
	WasmClientKeeper ibcwasmkeeper.Keeper
}
