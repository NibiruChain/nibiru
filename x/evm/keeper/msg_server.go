// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/evm"
)

var _ evm.MsgServer = &Keeper{}

func (k *Keeper) EthereumTx(
	goCtx context.Context, msg *evm.MsgEthereumTx,
) (resp *evm.MsgEthereumTxResponse, err error) {
	// TODO: feat(evm): EthereumTx impl
	return resp, common.ErrNotImplemented()
}

func (k *Keeper) UpdateParams(
	goCtx context.Context, msg *evm.MsgUpdateParams,
) (resp *evm.MsgUpdateParamsResponse, err error) {
	// TODO: feat(evm): UpdateParams impl
	return resp, common.ErrNotImplemented()
}
