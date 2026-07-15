package app

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdkclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis"
	crisistypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov"
	govclient "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/client"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/x/bank"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
)

// AuthModule extends the Cosmos SDK auth default genesis with the canonical
// ERC-2470 EthAccount required by the matching EVM genesis account.
type AuthModule struct {
	auth.AppModuleBasic
}

func (AuthModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	auth.AppModuleBasic{}.RegisterInterfaces(registry)
	eth.RegisterInterfaces(registry)
}

func (AuthModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := authtypes.DefaultGenesisState()
	accounts, err := authtypes.UnpackAccounts(genState.Accounts)
	if err != nil {
		panic(fmt.Errorf("failed to unpack auth default genesis accounts: %w", err))
	}

	for _, account := range accounts {
		if account.GetAddress().String() == evm.ERC2470Bech32Address {
			return cdc.MustMarshalJSON(genState)
		}
	}

	accounts = append(accounts, &eth.EthAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address:       evm.ERC2470Bech32Address,
			AccountNumber: uint64(len(accounts)),
			Sequence:      0,
		},
		CodeHash: evm.ERC2470CodeHash,
	})
	packed, err := authtypes.PackAccounts(accounts)
	if err != nil {
		panic(fmt.Errorf("failed to pack auth default genesis accounts: %w", err))
	}
	genState.Accounts = packed
	return cdc.MustMarshalJSON(genState)
}

// EnsureERC2470AuthAccount adds the canonical factory's auth account to a
// caller-supplied genesis when a test or tooling path replaces the default
// auth genesis wholesale.
func EnsureERC2470AuthAccount(cdc codec.JSONCodec, appState GenesisState) {
	var genState authtypes.GenesisState
	cdc.MustUnmarshalJSON(appState[authtypes.ModuleName], &genState)
	accounts, err := authtypes.UnpackAccounts(genState.Accounts)
	if err != nil {
		panic(fmt.Errorf("failed to unpack auth genesis accounts: %w", err))
	}
	for _, account := range accounts {
		if account.GetAddress().String() == evm.ERC2470Bech32Address {
			return
		}
	}
	nextAccountNumber := uint64(0)
	for _, account := range accounts {
		if account.GetAccountNumber() >= nextAccountNumber {
			nextAccountNumber = account.GetAccountNumber() + 1
		}
	}
	accounts = append(accounts, &eth.EthAccount{
		BaseAccount: &authtypes.BaseAccount{
			Address:       evm.ERC2470Bech32Address,
			AccountNumber: nextAccountNumber,
		},
		CodeHash: evm.ERC2470CodeHash,
	})
	genState.Accounts, err = authtypes.PackAccounts(accounts)
	if err != nil {
		panic(fmt.Errorf("failed to pack auth genesis accounts: %w", err))
	}
	appState[authtypes.ModuleName] = cdc.MustMarshalJSON(&genState)
}

// BankModule defines a custom wrapper around the x/bank module's AppModuleBasic
// implementation to provide custom default genesis state.
type BankModule struct {
	bank.AppModuleBasic
}

// DefaultGenesis returns custom Nibiru x/bank module genesis state.
func (BankModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	denomMetadata := banktypes.Metadata{
		Description: "The native staking token of the Nibiru network.",
		Base:        appconst.DENOM_UNIBI,
		Name:        DisplayDenom,
		Display:     DisplayDenom,
		Symbol:      DisplayDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    appconst.DENOM_UNIBI,
				Exponent: 0,
				Aliases: []string{
					"micronibi",
				},
			},
			{
				Denom:    DisplayDenom,
				Exponent: 6,
				Aliases:  []string{},
			},
		},
	}

	genState := banktypes.DefaultGenesisState()
	genState.DenomMetadata = append(genState.DenomMetadata, denomMetadata)
	return cdc.MustMarshalJSON(genState)
}

// StakingModule defines a custom wrapper around the x/staking module's
// AppModuleBasic implementation to provide custom default genesis state.
type StakingModule struct {
	staking.AppModuleBasic
}

var _ module.HasGenesisBasics = (*StakingModule)(nil)

// DefaultGenesis returns custom Nibiru x/staking module genesis state.
func (StakingModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := stakingtypes.DefaultGenesisState()
	genState.Params.BondDenom = appconst.DENOM_UNIBI
	genState.Params.MinCommissionRate = sdkmath.LegacyMustNewDecFromStr("0.05")
	return cdc.MustMarshalJSON(genState)
}

// ValidateGenesis: Verifies that the provided staking genesis state holds
// expected invariants. I.e., params in correct bounds, no duplicate validators.
// This implements the module.HasGenesisBasics interface and gets called during
// the setup of the chain's module.BasicManager.
func (StakingModule) ValidateGenesis(
	cdc codec.JSONCodec, txConfig sdkclient.TxEncodingConfig, bz json.RawMessage,
) error {
	gen := new(stakingtypes.GenesisState)
	if err := cdc.UnmarshalJSON(bz, gen); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %s", stakingtypes.ModuleName, bz)
	}
	if !gen.Params.MinCommissionRate.IsPositive() {
		return fmt.Errorf(
			"staking.params.min_commission must be positive (preferably >= 0.05): found value of %s",
			gen.Params.MinCommissionRate.String(),
		)
	}
	return staking.ValidateGenesis(gen)
}

// CrisisModule defines a custom wrapper around the x/crisis module's
// AppModuleBasic implementation to provide custom default genesis state.
type CrisisModule struct {
	crisis.AppModuleBasic
}

// DefaultGenesis returns custom Nibiru x/crisis module genesis state.
func (CrisisModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := crisistypes.DefaultGenesisState()
	genState.ConstantFee = sdk.NewCoin(appconst.DENOM_UNIBI, genState.ConstantFee.Amount)
	return cdc.MustMarshalJSON(genState)
}

// GovModule defines a custom wrapper around the x/gov module's
// AppModuleBasic implementation to provide custom default genesis state.
type GovModule struct {
	gov.AppModuleBasic
}

// DefaultGenesis returns custom Nibiru x/gov module genesis state.
func (GovModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := govtypes.DefaultGenesisState()
	genState.Params.MinDeposit = sdk.NewCoins(
		sdk.NewCoin(appconst.DENOM_UNIBI, govtypes.DefaultMinDepositTokens))
	return cdc.MustMarshalJSON(genState)
}

func NewGovModuleBasic(proposalHandlers ...govclient.ProposalHandler) GovModule {
	return GovModule{
		gov.NewAppModuleBasic(proposalHandlers),
	}
}
