package encoding_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth/encoding"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func TestTxEncoding(t *testing.T) {
	ethAcc := evmtest.NewEthAccInfo()
	addr, key := ethAcc.EthAddr, ethAcc.PrivKey
	signer := evmtest.NewSigner(key)

	ethTxParams := evm.EvmTxArgs{
		ChainID:   big.NewInt(1),
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  100000,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Input:     []byte{},
	}
	msg := evm.NewTx(&ethTxParams)
	msg.From = addr.Hex()

	ethSigner := gethcore.LatestSignerForChainID(big.NewInt(1))
	err := msg.Sign(ethSigner, signer)
	require.NoError(t, err)

	cfg := encoding.MakeConfig(app.ModuleBasics)

	_, err = cfg.TxConfig.TxEncoder()(msg)
	require.Error(t, err, "encoding failed")
}
