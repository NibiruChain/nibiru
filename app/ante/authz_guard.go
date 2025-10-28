// Copyright (c) 2023-2024 Nibi, Inc.
package ante

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

const maxNestedMsgs = 2

// AnteDecAuthzGuard filters autz messages
type AnteDecAuthzGuard struct{}

// AnteHandle rejects "authz grant generic --msg-type '/eth.evm.v1.MsgEthereumTx'"
// Also rejects authz exec tx.json with any MsgEthereumTx inside
func (anteDec AnteDecAuthzGuard) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		// Do not allow grant for MsgEthereumTx
		if msgGrant, ok := msg.(*authz.MsgGrant); ok {
			if msgGrant.Grant.Authorization == nil {
				return ctx, sdkioerrors.Wrapf(
					sdkerrors.ErrInvalidType,
					"grant authorization is missing",
				)
			}
			authorization, err := msgGrant.Grant.GetAuthorization()
			if err != nil {
				return ctx, sdkioerrors.Wrapf(
					sdkerrors.ErrInvalidType,
					"failed unmarshaling generic authorization %s", err,
				)
			}
			if genericAuth, ok := authorization.(*authz.GenericAuthorization); ok {
				if genericAuth.MsgTypeURL() == sdk.MsgTypeURL(&evm.MsgEthereumTx{}) {
					return ctx, sdkioerrors.Wrapf(
						sdkerrors.ErrNotSupported,
						"authz grant generic for msg type %s is not allowed",
						genericAuth.MsgTypeURL(),
					)
				}
			}
		}
		// Also reject MsgEthereumTx in exec
		if msgExec, ok := msg.(*authz.MsgExec); ok {
			if err := anteDec.checkMsgExecRecursively(msgExec, 0, maxNestedMsgs); err != nil {
				return ctx, sdkerrors.Wrapf(
					sdkerrors.ErrInvalidType,
					err.Error(),
				)
			}
		}
	}
	return next(ctx, tx, simulate)
}

func (anteDec AnteDecAuthzGuard) checkMsgExecRecursively(msgExec *authz.MsgExec, depth int, maxDepth int) error {
	if depth >= maxDepth {
		return fmt.Errorf("exceeded max nested message depth: %d", maxDepth)
	}

	msgsInExec, err := msgExec.GetMessages()
	if err != nil {
		return sdkerrors.Wrapf(
			sdkerrors.ErrInvalidType,
			"failed getting exec messages %s", err,
		)
	}

	for _, msg := range msgsInExec {
		if _, ok := msg.(*evm.MsgEthereumTx); ok {
			return sdkioerrors.Wrapf(
				sdkerrors.ErrInvalidType,
				"MsgEthereumTx needs to be contained within a tx with 'ExtensionOptionsEthereumTx' option",
			)
		}
		if nestedExec, ok := msg.(*authz.MsgExec); ok {
			if err := anteDec.checkMsgExecRecursively(nestedExec, depth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	return nil
}
