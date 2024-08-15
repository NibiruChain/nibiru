package evmtest

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func AssertERC20BalanceEqual(
	t *testing.T,
	deps TestDeps,
	erc20, account gethcommon.Address,
	expectedBalance *big.Int,
) {
	actualBalance, err := deps.EvmKeeper.ERC20().BalanceOf(erc20, account, deps.Ctx)
	assert.NoError(t, err)
	assert.Zero(t, expectedBalance.Cmp(actualBalance), "expected %s, got %s", expectedBalance, actualBalance)
}

// CreateFunTokenForBankCoin: Uses the "TestDeps.Sender" account to create a
// "FunToken" mapping for a new coin
func CreateFunTokenForBankCoin(
	deps *TestDeps, bankDenom string, s *suite.Suite,
) (funtoken evm.FunToken) {
	if deps.App.BankKeeper.HasDenomMetaData(deps.Ctx, bankDenom) {
		s.Failf("setting bank.DenomMetadata would overwrite existing denom \"%s\"", bankDenom)
	}

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

	deps.App.BankKeeper.SetDenomMetaData(deps.Ctx, bankMetadata)

	// Give the sender funds for the fee
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	s.T().Log("happy: CreateFunToken for the bank coin")
	createFuntokenResp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
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
	s.Equal(funtoken, createFuntokenResp.FuntokenMapping)

	s.T().Log("Expect ERC20 to be deployed")
	_, err = deps.EvmKeeper.Code(deps.Ctx,
		&evm.QueryCodeRequest{
			Address: erc20.String(),
		},
	)
	s.NoError(err)

	return funtoken
}
