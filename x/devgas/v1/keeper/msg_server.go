package keeper

import (
	"context"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

var _ types.MsgServer = &Keeper{}

// isContractCreatedFromFactory: Checks the smart contract info to see if it
// was created by the "factory". This condition is true if:
// (1) the info.Admin is the gov module
// (2) the info.Creator is another smart contract
// (3) the info.Admin is another contract
func (k Keeper) isContractCreatedFromFactory(
	ctx sdk.Context, info *wasmTypes.ContractInfo, msgSender sdk.AccAddress,
) bool {
	govMod := k.accountKeeper.GetModuleAddress(govtypes.ModuleName).String()
	switch {
	case info.Admin == govMod:
		// case 1: True if info.Admin is the gov module
		return true
	case len(info.Admin) == 0:
		// case 2: True if info.Creator is another smart contract
		creator, err := sdk.AccAddressFromBech32(info.Creator)
		if err != nil {
			return false
		}
		return k.wasmKeeper.HasContractInfo(ctx, creator)
	case info.Admin != msgSender.String():
		// case 3: True if info.Admin is another contract
		//
		// Note that if info.Admin == msgSender, then it's not a smart contract
		// because the signer for the TxMsg is always a non-contract account.
		admin, err := sdk.AccAddressFromBech32(info.Admin)
		if err != nil {
			return false
		}
		return k.wasmKeeper.HasContractInfo(ctx, admin)
	default:
		return false
	}
}

// GetContractAdminOrCreatorAddress ensures the deployer is the contract's
// admin OR creator if no admin is set for all msg_server feeshare functions.
func (k Keeper) GetContractAdminOrCreatorAddress(
	ctx sdk.Context, contract sdk.AccAddress, deployer string,
) (sdk.AccAddress, error) {
	// Ensure the deployer address is valid
	_, err := sdk.AccAddressFromBech32(deployer)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid deployer address %s", deployer)
	}

	// Retrieve contract info
	info := k.wasmKeeper.GetContractInfo(ctx, contract)
	if info == nil {
		return nil, sdkerrors.ErrUnauthorized.Wrapf(
			"contract with address %s not found in state", contract,
		)
	}

	// Check if the contract has an admin
	if len(info.Admin) == 0 {
		// No admin, so check if the deployer is the creator of the contract
		if info.Creator != deployer {
			return nil,
				sdkerrors.ErrUnauthorized.Wrapf("you are not the creator of this contract %s",
					info.Creator,
				)
		}

		contractCreator, err := sdk.AccAddressFromBech32(info.Creator)
		if err != nil {
			return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address %s", info.Creator)
		}

		// Deployer is the creator, return the controlling account as the creator's address
		return contractCreator, err
	}

	// Admin is set, so check if the deployer is the admin of the contract
	if info.Admin != deployer {
		return nil, sdkerrors.ErrUnauthorized.Wrapf(
			"you are not an admin of this contract %s", deployer,
		)
	}

	// Verify the admin address is valid
	contractAdmin, err := sdk.AccAddressFromBech32(info.Admin)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf(
			"%s: invalid admin address %s", err.Error(), info.Admin,
		)
	}

	return contractAdmin, err
}

// RegisterFeeShare registers a contract to receive transaction fees
func (k Keeper) RegisterFeeShare(
	goCtx context.Context,
	msg *types.MsgRegisterFeeShare,
) (*types.MsgRegisterFeeShareResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := k.GetParams(ctx)
	if !params.EnableFeeShare {
		return nil, types.ErrFeeShareDisabled
	}

	// Get Contract
	contract, err := sdk.AccAddressFromBech32(msg.ContractAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid contract address (%s)", err)
	}

	// Check if contract is already registered
	if k.IsFeeShareRegistered(ctx, contract) {
		return nil, types.ErrFeeShareAlreadyRegistered.Wrapf("contract is already registered %s", contract)
	}

	// Get the withdraw address of the contract
	withdrawer, err := sdk.AccAddressFromBech32(msg.WithdrawerAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid withdrawer address %s", msg.WithdrawerAddress)
	}

	// ensure msg.DeployerAddress is  valid
	msgSender, err := sdk.AccAddressFromBech32(msg.DeployerAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid deployer address %s", msg.DeployerAddress)
	}

	var deployer sdk.AccAddress

	if k.isContractCreatedFromFactory(ctx, k.wasmKeeper.GetContractInfo(ctx, contract), msgSender) {
		// Anyone is allowed to register the dev gas withdrawer for a smart
		// contract to be the contract itself, so long as the contract was
		// created from the "factory" (gov module or if contract admin or creator is another contract)
		if msg.WithdrawerAddress != msg.ContractAddress {
			return nil, types.ErrFeeShareInvalidWithdrawer.Wrapf(
				"withdrawer address must be the same as the contract address if it is from a "+
					"factory contract withdrawer: %s contract: %s",
				msg.WithdrawerAddress, msg.ContractAddress,
			)
		}

		// set the deployer address to the contract address so it can self register
		msg.DeployerAddress = msg.ContractAddress
		deployer, err = sdk.AccAddressFromBech32(msg.DeployerAddress)
		if err != nil {
			return nil, err
		}
	} else {
		// Check that the person who signed the message is the wasm contract
		// admin or creator (if no admin)
		deployer, err = k.GetContractAdminOrCreatorAddress(ctx, contract, msg.DeployerAddress)
		if err != nil {
			return nil, err
		}
	}

	// prevent storing the same address for deployer and withdrawer
	feeshare := types.NewFeeShare(contract, deployer, withdrawer)
	k.SetFeeShare(ctx, feeshare)

	k.Logger(ctx).Debug(
		"registering contract for transaction fees",
		"contract", msg.ContractAddress,
		"deployer", msg.DeployerAddress,
		"withdraw", msg.WithdrawerAddress,
	)

	return &types.MsgRegisterFeeShareResponse{}, ctx.EventManager().EmitTypedEvent(
		&types.EventRegisterDevGas{
			Deployer:   msg.DeployerAddress,
			Contract:   msg.ContractAddress,
			Withdrawer: msg.WithdrawerAddress,
		},
	)
}

// UpdateFeeShare updates the withdraw address of a given FeeShare. If the given
// withdraw address is empty or the same as the deployer address, the withdraw
// address is removed.
func (k Keeper) UpdateFeeShare(
	goCtx context.Context,
	msg *types.MsgUpdateFeeShare,
) (*types.MsgUpdateFeeShareResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if !params.EnableFeeShare {
		return nil, types.ErrFeeShareDisabled
	}

	contract, err := sdk.AccAddressFromBech32(msg.ContractAddress)
	if err != nil {
		return nil,
			sdkerrors.ErrInvalidAddress.Wrapf(
				"invalid contract address (%s)", err,
			)
	}

	feeshare, found := k.GetFeeShare(ctx, contract)
	if !found {
		return nil,
			types.ErrFeeShareContractNotRegistered.Wrapf(
				"contract %s is not registered", msg.ContractAddress,
			)
	}

	// feeshare with the given withdraw address is already registered
	if msg.WithdrawerAddress == feeshare.WithdrawerAddress {
		return nil, types.ErrFeeShareAlreadyRegistered.Wrapf(
			"feeshare with withdraw address %s is already registered", msg.WithdrawerAddress,
		)
	}

	// Check that the person who signed the message is the wasm contract admin, if so return the deployer address
	_, err = k.GetContractAdminOrCreatorAddress(ctx, contract, msg.DeployerAddress)
	if err != nil {
		return nil, err
	}

	newWithdrawAddr, err := sdk.AccAddressFromBech32(msg.WithdrawerAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid WithdrawerAddress %s", msg.WithdrawerAddress,
		)
	}

	// update feeshare with new withdrawer
	feeshare.WithdrawerAddress = newWithdrawAddr.String()
	k.SetFeeShare(ctx, feeshare)

	return &types.MsgUpdateFeeShareResponse{}, ctx.EventManager().EmitTypedEvent(
		&types.EventUpdateDevGas{
			Deployer:   msg.DeployerAddress,
			Contract:   msg.ContractAddress,
			Withdrawer: msg.WithdrawerAddress,
		},
	)
}

// CancelFeeShare deletes the FeeShare for a given contract
func (k Keeper) CancelFeeShare(
	goCtx context.Context,
	msg *types.MsgCancelFeeShare,
) (*types.MsgCancelFeeShareResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := k.GetParams(ctx)
	if !params.EnableFeeShare {
		return nil, types.ErrFeeShareDisabled
	}

	contract, err := sdk.AccAddressFromBech32(msg.ContractAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid contract address (%s)", err)
	}

	fee, found := k.GetFeeShare(ctx, contract)
	if !found {
		return nil, types.ErrFeeShareContractNotRegistered.Wrapf(
			"contract %s is not registered", msg.ContractAddress,
		)
	}

	// Check that the person who signed the message is the wasm contract admin, if so return the deployer address
	_, err = k.GetContractAdminOrCreatorAddress(ctx, contract, msg.DeployerAddress)
	if err != nil {
		return nil, err
	}

	err = k.DevGasStore.Delete(ctx, fee.GetContractAddress())
	if err != nil {
		return nil, err
	}

	return &types.MsgCancelFeeShareResponse{}, ctx.EventManager().EmitTypedEvent(
		&types.EventCancelDevGas{
			Deployer: msg.DeployerAddress,
			Contract: msg.ContractAddress,
		},
	)
}

func (k Keeper) UpdateParams(
	goCtx context.Context, req *types.MsgUpdateParams,
) (resp *types.MsgUpdateParamsResponse, err error) {
	if k.authority != req.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := req.Params.Validate(); err != nil {
		return resp, err
	}
	k.ModuleParams.Set(ctx, req.Params)

	return &types.MsgUpdateParamsResponse{}, err
}
