// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmstate "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
)

var _ AnteStep = EthSigVerification

// AnteHandle validates checks that the registered chain id is the same as the
// one on the message, and that the signer address matches the one defined on the
// message. It's not skipped for RecheckTx, because it sets `From` address which
// is critical from other ante handler to work. Failure in RecheckTx will prevent
// tx to be included into block, especially when CheckTx succeed, in which case
// user won't see the error message.
func EthSigVerification(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) (err error) {
	chainID := k.EthChainID(sdb.Ctx())
	ethCfg := evm.EthereumConfig(chainID)
	blockNum := big.NewInt(sdb.Ctx().BlockHeight())
	signer := gethcore.MakeSigner(
		ethCfg,
		blockNum,
		evm.ParseBlockTimeUnixU64(sdb.Ctx()),
	)

	ethTx, err := msgEthTx.AsTransactionSafe()
	if err != nil {
		return err
	}

	sender, err := signer.Sender(ethTx)
	if err != nil {
		return sdkioerrors.Wrapf(
			sdkerrors.ErrorInvalidSigner,
			"couldn't retrieve sender address from the ethereum transaction: %s",
			err.Error(),
		)
	}

	// set up the sender to the transaction field if not already
	msgEthTx.From = sender.Hex()

	return nil
}
