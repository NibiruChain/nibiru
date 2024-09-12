// Copyright (c) 2023-2024 Nibi, Inc.
package ante

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/gogoproto/proto"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var genericAuthTypeTurl = "/" + proto.MessageName(&authz.GenericAuthorization{})

// AnteDecoratorAuthzGuard filters autz messages
type AnteDecoratorAuthzGuard struct{}

// AnteHandle rejects "authz grant generic --msg-type '/eth.evm.v1.MsgEthereumTx'"
func (rmd AnteDecoratorAuthzGuard) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		if msgGrant, ok := msg.(*authz.MsgGrant); ok {
			if msgGrant.Grant.Authorization == nil {
				return ctx, errors.Wrapf(
					errortypes.ErrInvalidType,
					"grant authorization is missing",
				)
			}
			if msgGrant.Grant.Authorization.TypeUrl == genericAuthTypeTurl {
				var genericAuth authz.GenericAuthorization
				err = proto.Unmarshal(msgGrant.Grant.Authorization.Value, &genericAuth)
				if err != nil {
					return ctx, errors.Wrapf(
						errortypes.ErrInvalidType,
						"failed unmarshaling generic authorization",
					)
				}
				if genericAuth.MsgTypeURL() == sdk.MsgTypeURL(&evm.MsgEthereumTx{}) {
					return ctx, errors.Wrapf(
						errortypes.ErrNotSupported,
						"authz grant generic for msg type %s is not allowed",
						genericAuth.MsgTypeURL(),
					)
				}
			}
		}
	}
	return next(ctx, tx, simulate)
}
