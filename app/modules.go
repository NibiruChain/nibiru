package app

import (
	"cosmossdk.io/math"
	"encoding/json"
	"fmt"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BankModule defines a custom wrapper around the x/bank module's AppModuleBasic
// implementation to provide custom default genesis state.
type BankModule struct {
	bank.AppModuleBasic
}

// DefaultGenesis returns custom Nibiru x/bank module genesis state.
func (BankModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	denomMetadata := banktypes.Metadata{
		Description: "The native staking token of the Nibiru network.",
		Base:        BondDenom,
		Name:        DisplayDenom,
		Display:     DisplayDenom,
		Symbol:      DisplayDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    BondDenom,
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
	genState.Params.BondDenom = BondDenom
	genState.Params.MinCommissionRate = math.LegacyMustNewDecFromStr("0.05")
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
	genState.ConstantFee = sdk.NewCoin(BondDenom, genState.ConstantFee.Amount)
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
		sdk.NewCoin(BondDenom, govtypes.DefaultMinDepositTokens))
	return cdc.MustMarshalJSON(genState)
}

func NewGovModuleBasic(proposalHandlers ...govclient.ProposalHandler) GovModule {
	return GovModule{
		gov.NewAppModuleBasic(proposalHandlers),
	}
}
