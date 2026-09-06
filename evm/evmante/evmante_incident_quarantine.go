package evmante

import (
	sdkioerrors "cosmossdk.io/errors"

	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
)

// AnteStepIncidentQuarantine rejects authenticated EVM transactions from an
// account quarantined by the v2.18.1 mainnet incident response. This step must
// run after EthSigVerification, which recovers and sets MsgEthereumTx.From.
func AnteStepIncidentQuarantine(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) error {
	sender := msgEthTx.FromAddrBech32()
	if ante.IsIncidentQuarantinedAddress(sdb.Ctx(), sender) {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrUnauthorized,
			"account %s is quarantined",
			sender.String(),
		)
	}
	return nil
}
