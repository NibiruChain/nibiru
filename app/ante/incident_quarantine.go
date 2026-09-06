package ante

import (
	"bytes"

	sdkioerrors "cosmossdk.io/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
)

// incidentQuarantinedAddresses contains the account identities quarantined by
// the v2.18.1 mainnet incident response. Cosmos and EVM addresses share the
// same 20 address bytes, so callers must normalize to sdk.AccAddress before
// checking this list.
var incidentQuarantinedAddresses = []sdk.AccAddress{
	sdk.AccAddress(gethcommon.HexToAddress("0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B").Bytes()),
}

// IsIncidentQuarantinedAddress reports whether addr is barred from exercising
// account authority on Nibiru mainnet. The chain-ID check prevents the
// incident-specific policy from affecting testnets and local development.
func IsIncidentQuarantinedAddress(ctx sdk.Context, addr sdk.AccAddress) bool {
	if ctx.ChainID() != appconst.SDK_CHAIN_ID_MAINNET {
		return false
	}

	for _, quarantined := range incidentQuarantinedAddresses {
		if bytes.Equal(addr, quarantined) {
			return true
		}
	}
	return false
}

// AnteDecIncidentQuarantine rejects Cosmos transactions that exercise a
// quarantined account's authority. It checks each message signer and descends
// into MsgExec because the outer signer is the grantee while the inner message
// signer is the effective granter.
type AnteDecIncidentQuarantine struct{}

func (AnteDecIncidentQuarantine) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		feeGranter := feeTx.FeeGranter()
		if !feeGranter.Empty() && IsIncidentQuarantinedAddress(ctx, feeGranter) {
			return ctx, sdkioerrors.Wrapf(
				sdkerrors.ErrUnauthorized,
				"account %s is quarantined and cannot grant fees",
				feeGranter.String(),
			)
		}
	}

	for _, msg := range tx.GetMsgs() {
		if err := rejectQuarantinedMessage(ctx, msg, 0); err != nil {
			return ctx, err
		}
	}
	return next(ctx, tx, simulate)
}

func rejectQuarantinedMessage(ctx sdk.Context, msg sdk.Msg, depth int) error {
	for _, signer := range msg.GetSigners() {
		if IsIncidentQuarantinedAddress(ctx, signer) {
			return sdkioerrors.Wrapf(
				sdkerrors.ErrUnauthorized,
				"account %s is quarantined",
				signer.String(),
			)
		}
	}

	msgExec, ok := msg.(*authz.MsgExec)
	if !ok {
		return nil
	}
	if depth >= maxNestedMsgs {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInvalidType,
			"exceeded max nested message depth: %d",
			maxNestedMsgs,
		)
	}

	innerMsgs, err := msgExec.GetMessages()
	if err != nil {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrInvalidType,
			"failed getting exec messages %s",
			err,
		)
	}
	for _, innerMsg := range innerMsgs {
		if err := rejectQuarantinedMessage(ctx, innerMsg, depth+1); err != nil {
			return err
		}
	}
	return nil
}
