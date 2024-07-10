package evmtest

import (
	"math/big"
	"testing"

	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

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
	deps TestDeps,
	contract, account gethcommon.Address,
	balance *big.Int,
) {
	gotBalance, err := deps.K.ERC20().BalanceOf(contract, account, deps.Ctx)
	assert.NoError(t, err)
	assert.Equal(t, balance.String(), gotBalance.String())
}

// CreateFunTokenForBankCoin: Uses the "TestDeps.Sender" account to create a
// "FunToken" mapping for a new coin
func CreateFunTokenForBankCoin(
	deps *TestDeps, bankDenom string, s *suite.Suite,
) (funtoken evm.FunToken) {
	s.T().Log("Setup: Create a coin in the bank state")
	bankMetadata := bank.Metadata{
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
		},
		Base:    bankDenom,
		Display: bankDenom,
		Name:    bankDenom,
		Symbol:  bankDenom,
	}
	if deps.Chain.BankKeeper.HasDenomMetaData(deps.Ctx, bankDenom) {
		s.Failf("setting bank.DenomMetadata would overwrite existing denom \"%s\"", bankDenom)
	}
	deps.Chain.BankKeeper.SetDenomMetaData(deps.Ctx, bankMetadata)

	s.T().Log("happy: CreateFunToken for the bank coin")
	createFuntokenResp, err := deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.NoError(err, "bankDenom %s", bankDenom)
	erc20 := createFuntokenResp.FuntokenMapping.Erc20Addr
	funtoken = evm.FunToken{
		Erc20Addr:      erc20,
		BankDenom:      bankDenom,
		IsMadeFromCoin: true,
	}
	s.Equal(createFuntokenResp.FuntokenMapping, funtoken)

	s.T().Log("Expect ERC20 to be deployed")
	erc20Addr := erc20.ToAddr()
	queryCodeReq := &evm.QueryCodeRequest{
		Address: erc20Addr.String(),
	}
	_, err = deps.K.Code(deps.Ctx, queryCodeReq)
	s.NoError(err)

	return funtoken
}
