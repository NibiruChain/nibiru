package evmtest

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func AssertERC20BalanceEqual(
	t *testing.T,
	deps TestDeps,
	erc20, account gethcommon.Address,
	expectedBalance *big.Int,
) {
	AssertERC20BalanceEqualWithDescription(t, deps, erc20, account, expectedBalance, "")
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

func AssertBankBalanceEqual(
	t *testing.T,
	deps TestDeps,
	denom string,
	account gethcommon.Address,
	expectedBalance *big.Int,
) {
	AssertBankBalanceEqualWithDescription(
		t, deps, denom, account, expectedBalance, "",
	)
}

// BigPow multiplies "amount" by 10 to the "pow10Exp".
func BigPow(amount *big.Int, pow10Exp uint8) (powAmount *big.Int) {
	pow10 := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(pow10Exp)), nil)
	return new(big.Int).Mul(amount, pow10)
}

type FunTokenBalanceAssert struct {
	FunToken     evm.FunToken
	Account      gethcommon.Address
	BalanceBank  *big.Int
	BalanceERC20 *big.Int
	Description  string
}

func (bals FunTokenBalanceAssert) Assert(t *testing.T, deps TestDeps) {
	AssertERC20BalanceEqualWithDescription(
		t, deps, bals.FunToken.Erc20Addr.Address, bals.Account, bals.BalanceERC20,
		bals.Description,
	)
	AssertBankBalanceEqualWithDescription(
		t, deps, bals.FunToken.BankDenom, bals.Account, bals.BalanceBank,
		bals.Description,
	)
}

func AssertERC20BalanceEqualWithDescription(
	t *testing.T,
	deps TestDeps,
	erc20, account gethcommon.Address,
	expectedBalance *big.Int,
	description string,
) {
	actualBalance, err := deps.EvmKeeper.ERC20().BalanceOf(erc20, account, deps.Ctx)
	var errSuffix string
	if description == "" {
		errSuffix = description
	} else {
		errSuffix = ": " + description
	}
	assert.NoError(t, err, errSuffix)
	assert.Equalf(t, expectedBalance.String(), actualBalance.String(),
		"expected %s, got %s", expectedBalance, actualBalance,
		errSuffix,
	)
}

func AssertBankBalanceEqualWithDescription(
	t *testing.T,
	deps TestDeps,
	denom string,
	account gethcommon.Address,
	expectedBalance *big.Int,
	description string,
) {
	bech32Addr := eth.EthAddrToNibiruAddr(account)
	actualBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, bech32Addr, denom).Amount.BigInt()
	var errSuffix string
	if description == "" {
		errSuffix = description
	} else {
		errSuffix = ": " + description
	}
	assert.Equalf(t, expectedBalance.String(), actualBalance.String(),
		"expected %s, got %s", expectedBalance, actualBalance, errSuffix)
}

const (
	// FunTokenGasLimitSendToEvm consists of gas for 3 calls:
	// 1. transfer erc20 from sender to module
	//    ~60_000 gas for regular erc20 transfer (our own ERC20Minter contract)
	//    could be higher for user created contracts, let's cap with 200_000
	// 2. mint native coin (made from erc20) or burn erc20 token (made from coin)
	//	  ~60_000 gas for either mint or burn
	// 3. send from module to account:
	//	  ~65_000 gas (bank send)
	FunTokenGasLimitSendToEvm uint64 = 400_000
)
