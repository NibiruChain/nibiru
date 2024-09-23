// Copyright (c) 2023-2024 Nibi, Inc.
package ante

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// AnteDecoratorAuthzGuard filters autz messages
type AnteDecoratorAuthzGuard struct{}

// AnteHandle rejects "authz grant generic --msg-type '/eth.evm.v1.MsgEthereumTx'"
// Also rejects authz exec tx.json with any MsgEthereumTx inside
func (rmd AnteDecoratorAuthzGuard) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		// Do not allow grant for MsgEthereumTx
		if msgGrant, ok := msg.(*authz.MsgGrant); ok {
			if msgGrant.Grant.Authorization == nil {
				return ctx, errors.Wrapf(
					errortypes.ErrInvalidType,
					"grant authorization is missing",
				)
			}
			authorization, err := msgGrant.Grant.GetAuthorization()
			if err != nil {
				return ctx, errors.Wrapf(
					errortypes.ErrInvalidType,
					"failed unmarshaling generic authorization %s", err,
				)
			}
			if genericAuth, ok := authorization.(*authz.GenericAuthorization); ok {
				if genericAuth.MsgTypeURL() == sdk.MsgTypeURL(&evm.MsgEthereumTx{}) {
					return ctx, errors.Wrapf(
						errortypes.ErrNotSupported,
						"authz grant generic for msg type %s is not allowed",
						genericAuth.MsgTypeURL(),
					)
				}
			}
		}
		// Also reject MsgEthereumTx in exec
		if msgExec, ok := msg.(*authz.MsgExec); ok {
			msgsInExec, err := msgExec.GetMessages()
			if err != nil {
				return ctx, errors.Wrapf(
					errortypes.ErrInvalidType,
					"failed getting exec messages %s", err,
				)
			}
			for _, msgInExec := range msgsInExec {
				if _, ok := msgInExec.(*evm.MsgEthereumTx); ok {
					return ctx, errors.Wrapf(
						errortypes.ErrInvalidType,
						"MsgEthereumTx needs to be contained within a tx with 'ExtensionOptionsEthereumTx' option",
					)
				}
			}
		}
	}
	return next(ctx, tx, simulate)
}
