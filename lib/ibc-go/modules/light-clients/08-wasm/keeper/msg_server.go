package keeper

import (
	"context"
	"encoding/hex"

	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/errors"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/08-wasm/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// StoreCode defines a rpc handler method for MsgStoreCode
func (k Keeper) StoreCode(goCtx context.Context, msg *types.MsgStoreCode) (*types.MsgStoreCodeResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, sdkioerrors.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	checksum, err := k.storeWasmCode(ctx, msg.WasmByteCode, ibcwasm.GetVM().StoreCode)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to store wasm bytecode")
	}

	emitStoreWasmCodeEvent(ctx, checksum)

	return &types.MsgStoreCodeResponse{
		Checksum: checksum,
	}, nil
}

// RemoveChecksum defines a rpc handler method for MsgRemoveChecksum
func (k Keeper) RemoveChecksum(goCtx context.Context, msg *types.MsgRemoveChecksum) (*types.MsgRemoveChecksumResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, sdkioerrors.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if !types.HasChecksum(ctx, k.cdc, msg.Checksum) {
		return nil, types.ErrWasmChecksumNotFound
	}

	err := types.RemoveChecksum(ctx, k.cdc, ibcwasm.GetWasmStoreKey(), msg.Checksum)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to remove checksum")
	}

	// unpin the code from the vm in-memory cache
	if err := ibcwasm.GetVM().Unpin(msg.Checksum); err != nil {
		return nil, sdkioerrors.Wrapf(err, "failed to unpin contract with checksum (%s) from vm cache", hex.EncodeToString(msg.Checksum))
	}

	return &types.MsgRemoveChecksumResponse{}, nil
}

// MigrateContract defines a rpc handler method for MsgMigrateContract
func (k Keeper) MigrateContract(goCtx context.Context, msg *types.MsgMigrateContract) (*types.MsgMigrateContractResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, sdkioerrors.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.migrateContractCode(ctx, msg.ClientId, msg.Checksum, msg.Msg)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to migrate contract")
	}

	// event emission is handled in migrateContractCode

	return &types.MsgMigrateContractResponse{}, nil
}
