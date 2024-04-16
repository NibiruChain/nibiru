package encoding_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth/encoding"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	evmtypes "github.com/NibiruChain/nibiru/x/evm/types"
)

func TestTxEncoding(t *testing.T) {
	addr, key := testutil.PrivKeyEth()
	signer := testutil.NewSigner(key)

	ethTxParams := evmtypes.EvmTxArgs{
		ChainID:   big.NewInt(1),
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  100000,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Input:     []byte{},
	}
	msg := evmtypes.NewTx(&ethTxParams)
	msg.From = addr.Hex()

	ethSigner := ethtypes.LatestSignerForChainID(big.NewInt(1))
	err := msg.Sign(ethSigner, signer)
	require.NoError(t, err)

	cfg := encoding.MakeConfig(app.ModuleBasics)

	_, err = cfg.TxConfig.TxEncoder()(msg)
	require.Error(t, err, "encoding failed")
}
