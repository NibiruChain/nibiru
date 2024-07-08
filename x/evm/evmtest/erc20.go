package evmtest

import (
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/evm"
)

func DoEthTx(
	deps *TestDeps, contract, from gethcommon.Address, input []byte,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	commit := true
	return deps.K.CallContractWithInput(
		deps.Ctx, from, &contract, commit, input,
	)
}

func AssertERC20BalanceEqual(
	t *testing.T,
	deps *TestDeps,
	contract, account gethcommon.Address,
	balance *big.Int,
) {
	gotBalance, err := deps.K.ERC20().BalanceOf(contract, account, deps.Ctx)
	assert.NoError(t, err)
	assert.Equal(t, balance.String(), gotBalance.String())
}
