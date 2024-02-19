package simulation

import (
	"math/rand"

	"github.com/NibiruChain/nibiru/x/tokenfactory/keeper"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// Simulation parameter constants
const (
	StakePerAccount           = "stake_per_account"
	InitiallyBondedValidators = "initially_bonded_validators"

	DefaultWeightMsgCreateDenom      int = 100
	DefaultWeightMsgMint             int = 100
	DefaultWeightMsgBurn             int = 100
	DefaultWeightMsgChangeAdmin      int = 100
	DefaultWeightMsgSetDenomMetadata int = 100
)

// Simulation operation weights constants
//
//nolint:gosec
const (
	OpWeightMsgCreateDenom      = "op_weight_msg_create_denom"
	OpWeightMsgChangeAdmin      = "op_weight_msg_change_admin"
	OpWeightMsgMint             = "op_weight_msg_mint"
	OpWeightMsgBurn             = "op_weight_msg_burn"
	OpWeightMsgSetDenomMetadata = "op_weight_msg_set_denom_metadata"
)

type BankKeeper interface {
	simulation.BankKeeper
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// BuildOperationInput helper to build object
func BuildOperationInput(
	r *rand.Rand,
	app *baseapp.BaseApp,
	ctx sdk.Context,
	msg interface {
		sdk.Msg
		Type() string
	},
	simAccount simtypes.Account,
	ak types.AccountKeeper,
	bk BankKeeper,
	deposit sdk.Coins,
) simulation.OperationInput {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	txConfig := tx.NewTxConfig(codec.NewProtoCodec(interfaceRegistry), tx.DefaultSignModes)
	return simulation.OperationInput{
		R:               r,
		App:             app,
		TxGen:           txConfig,
		Cdc:             nil,
		Msg:             msg,
		MsgType:         msg.Type(),
		Context:         ctx,
		SimAccount:      simAccount,
		AccountKeeper:   ak,
		Bankkeeper:      bk,
		ModuleName:      types.ModuleName,
		CoinsSpentInMsg: deposit,
	}
}

func WeightedOperations(
	simstate *module.SimulationState,
	tfKeeper keeper.Keeper,
	ak types.AccountKeeper,
	bk BankKeeper,
) simulation.WeightedOperations {

	var (
		weightMsgCreateDenom      int
		weightMsgChangeAdmin      int
		weightMsgMint             int
		weightMsgBurn             int
		weightMsgSetDenomMetadata int
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgCreateDenom, &weightMsgCreateDenom, nil,
		func(_ *rand.Rand) {
			weightMsgCreateDenom = DefaultWeightMsgCreateDenom
		},
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgChangeAdmin, &weightMsgChangeAdmin, nil,
		func(_ *rand.Rand) {
			weightMsgChangeAdmin = DefaultWeightMsgChangeAdmin
		},
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgMint, &weightMsgMint, nil,
		func(_ *rand.Rand) {
			weightMsgMint = DefaultWeightMsgMint
		},
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgBurn, &weightMsgBurn, nil,
		func(_ *rand.Rand) {
			weightMsgBurn = DefaultWeightMsgBurn
		},
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgSetDenomMetadata, &weightMsgSetDenomMetadata, nil,
		func(_ *rand.Rand) {
			weightMsgSetDenomMetadata = DefaultWeightMsgSetDenomMetadata
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgCreateDenom,
			SimulateMsgCreateDenom(
				tfKeeper,
				ak,
				bk,
			),
		),
		simulation.NewWeightedOperation(
			weightMsgChangeAdmin,
			SimulateMsgChangeAdmin(
				tfKeeper,
				ak,
				bk,
				DefaultSimulationDenomSelector,
			),
		),
		simulation.NewWeightedOperation(
			weightMsgMint,
			SimulateMsgMint(
				tfKeeper,
				ak,
				bk,
				DefaultSimulationDenomSelector,
			),
		),
		simulation.NewWeightedOperation(
			weightMsgBurn,
			SimulateMsgBurn(
				tfKeeper,
				ak,
				bk,
				DefaultSimulationDenomSelector,
			),
		),
		simulation.NewWeightedOperation(
			weightMsgSetDenomMetadata,
			SimulateMsgSetDenomMetadata(
				tfKeeper,
				ak,
				bk,
				DefaultSimulationDenomSelector,
			),
		),
	}
}

type DenomSelector = func(*rand.Rand, sdk.Context, keeper.Keeper, string) (string, bool)

func DefaultSimulationDenomSelector(r *rand.Rand, ctx sdk.Context, tfKeeper keeper.Keeper, creator string) (string, bool) {
	denoms := tfKeeper.QueryDenoms(ctx, creator)
	if len(denoms) == 0 {
		return "", false
	}
	randPos := r.Intn(len(denoms))

	return denoms[randPos], true
}

// Simulate msg create denom
func SimulateMsgCreateDenom(
	tfKeeper keeper.Keeper,
	ak types.AccountKeeper,
	bk BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get sims account
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// TODO: Check if sims account enough create fee when CreateDenom Msg charge

		// Create msg create denom
		msg := types.MsgCreateDenom{
			Sender:   simAccount.Address.String(),
			Subdenom: simtypes.RandStringOfLength(r, 10),
		}

		txCtx := BuildOperationInput(r, app, ctx, &msg, simAccount, ak, bk, nil)
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

func SimulateMsgChangeAdmin(
	tfKeeper keeper.Keeper,
	ak types.AccountKeeper,
	bk BankKeeper,
	denomSelector DenomSelector,
) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get create denom account
		createdDenomAccount, _ := simtypes.RandomAcc(r, accs)

		// Get demon
		denom, hasDenom := denomSelector(r, ctx, tfKeeper, createdDenomAccount.Address.String())
		if !hasDenom {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgChangeAdmin{}.Type(), "sim account have no denom created"), nil, nil
		}

		// Get admin of the denom
		authData, err := tfKeeper.Store.GetDenomAuthorityMetadata(ctx, denom)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgChangeAdmin{}.Type(), "err authority metadata"), nil, err
		}
		curAdminAccount, found := simtypes.FindAccount(accs, sdk.MustAccAddressFromBech32(authData.Admin))
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgChangeAdmin{}.Type(), "admin account not found"), nil, nil
		}

		// Rand new admin account
		newAdmin, _ := simtypes.RandomAcc(r, accs)
		if newAdmin.Address.String() == curAdminAccount.Address.String() {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgChangeAdmin{}.Type(), "new admin cannot be the same as current admin"), nil, nil
		}

		// Create msg
		msg := types.MsgChangeAdmin{
			Sender:   curAdminAccount.Address.String(),
			Denom:    denom,
			NewAdmin: newAdmin.Address.String(),
		}

		txCtx := BuildOperationInput(r, app, ctx, &msg, curAdminAccount, ak, bk, nil)
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// Simulate msg mint denom
func SimulateMsgMint(
	tfKeeper keeper.Keeper,
	ak types.AccountKeeper,
	bk BankKeeper,
	denomSelector DenomSelector,
) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get create denom account
		createdDenomAccount, _ := simtypes.RandomAcc(r, accs)

		// Get demon
		denom, hasDenom := denomSelector(r, ctx, tfKeeper, createdDenomAccount.Address.String())
		if !hasDenom {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgMint{}.Type(), "sim account have no denom created"), nil, nil
		}

		// Get admin of the denom
		authData, err := tfKeeper.Store.GetDenomAuthorityMetadata(ctx, denom)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgMint{}.Type(), "err authority metadata"), nil, err
		}
		adminAccount, found := simtypes.FindAccount(accs, sdk.MustAccAddressFromBech32(authData.Admin))
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgMint{}.Type(), "admin account not found"), nil, nil
		}

		// Rand mint amount
		mintAmount, _ := simtypes.RandPositiveInt(r, sdk.NewIntFromUint64(100_000_000))

		// Create msg mint
		msg := types.MsgMint{
			Sender: adminAccount.Address.String(),
			Coin:   sdk.NewCoin(denom, mintAmount),
		}

		txCtx := BuildOperationInput(r, app, ctx, &msg, adminAccount, ak, bk, nil)
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

func SimulateMsgBurn(
	tfKeeper keeper.Keeper,
	ak types.AccountKeeper,
	bk BankKeeper,
	denomSelector DenomSelector,
) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get create denom account
		createdDenomAccount, _ := simtypes.RandomAcc(r, accs)

		// Get demon
		denom, hasDenom := denomSelector(r, ctx, tfKeeper, createdDenomAccount.Address.String())
		if !hasDenom {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgBurn{}.Type(), "sim account have no denom created"), nil, nil
		}

		// Get admin of the denom
		authData, err := tfKeeper.Store.GetDenomAuthorityMetadata(ctx, denom)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgBurn{}.Type(), "err authority metadata"), nil, err
		}
		adminAccount, found := simtypes.FindAccount(accs, sdk.MustAccAddressFromBech32(authData.Admin))
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgBurn{}.Type(), "admin account not found"), nil, nil
		}

		// Check if admin account balance = 0
		accountBalance := bk.GetBalance(ctx, adminAccount.Address, denom)
		if accountBalance.Amount.LTE(sdk.ZeroInt()) {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgBurn{}.Type(), "sim account have no balance"), nil, nil
		}

		// Rand burn amount
		amount, _ := simtypes.RandPositiveInt(r, accountBalance.Amount)
		burnAmount := sdk.NewCoin(denom, amount)

		// Create msg
		msg := types.MsgBurn{
			Sender:   adminAccount.Address.String(),
			Coin:     burnAmount,
			BurnFrom: adminAccount.Address.String(),
		}

		txCtx := BuildOperationInput(r, app, ctx, &msg, adminAccount, ak, bk, sdk.NewCoins(burnAmount))
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

func SimulateMsgSetDenomMetadata(
	tfKeeper keeper.Keeper,
	ak types.AccountKeeper,
	bk BankKeeper,
	denomSelector DenomSelector,
) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get create denom account
		createdDenomAccount, _ := simtypes.RandomAcc(r, accs)

		// Get demon
		denom, hasDenom := denomSelector(r, ctx, tfKeeper, createdDenomAccount.Address.String())
		if !hasDenom {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgSetDenomMetadata{}.Type(), "sim account have no denom created"), nil, nil
		}

		// Get admin of the denom
		authData, err := tfKeeper.Store.GetDenomAuthorityMetadata(ctx, denom)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgSetDenomMetadata{}.Type(), "err authority metadata"), nil, err
		}
		adminAccount, found := simtypes.FindAccount(accs, sdk.MustAccAddressFromBech32(authData.Admin))
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgSetDenomMetadata{}.Type(), "admin account not found"), nil, nil
		}

		metadata := banktypes.Metadata{
			Description: simtypes.RandStringOfLength(r, 10),
			DenomUnits: []*banktypes.DenomUnit{{
				Denom:    denom,
				Exponent: 0,
			}},
			Base:    denom,
			Display: denom,
			Name:    simtypes.RandStringOfLength(r, 10),
			Symbol:  simtypes.RandStringOfLength(r, 10),
		}

		msg := types.MsgSetDenomMetadata{
			Sender:   adminAccount.Address.String(),
			Metadata: metadata,
		}

		txCtx := BuildOperationInput(r, app, ctx, &msg, adminAccount, ak, bk, nil)
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
