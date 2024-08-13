package evmtest

import (
	"math/big"
	"testing"

	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func DoEthTx(
	deps *TestDeps, contract, from gethcommon.Address, input []byte,
) (evmResp *evm.MsgEthereumTxResponse, err error) {
	commit := true
	return deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, from, &contract, commit, input,
	)
}

func AssertERC20BalanceEqual(
	t *testing.T,
	deps TestDeps,
	contract, account gethcommon.Address,
	balance *big.Int,
) {
	gotBalance, err := deps.EvmKeeper.ERC20().BalanceOf(contract, account, deps.Ctx)
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
	if deps.App.BankKeeper.HasDenomMetaData(deps.Ctx, bankDenom) {
		s.Failf("setting bank.DenomMetadata would overwrite existing denom \"%s\"", bankDenom)
	}
	deps.App.BankKeeper.SetDenomMetaData(deps.Ctx, bankMetadata)

	// Give the sender funds for the fee
	err := testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	)
	s.Require().NoError(err)

	s.T().Log("happy: CreateFunToken for the bank coin")
	createFuntokenResp, err := deps.EvmKeeper.CreateFunToken(
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
	_, err = deps.EvmKeeper.Code(deps.Ctx, queryCodeReq)
	s.NoError(err)

	return funtoken
}
