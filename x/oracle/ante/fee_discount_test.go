package ante_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	testapp "github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/oracle/ante"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

// MockStakingKeeperI implements ante.StakingKeeperI
type MockStakingKeeperI struct {
	validators  map[string]stakingtypes.Validator
	totalBonded sdk.Int
}

func (m MockStakingKeeperI) GetValidator(ctx sdk.Context, addr sdk.ValAddress) (stakingtypes.Validator, bool) {
	v, ok := m.validators[addr.String()]
	return v, ok
}
func (m MockStakingKeeperI) TotalBondedTokens(ctx sdk.Context) sdk.Int {
	return m.totalBonded
}

// MockOracleKeeperI implements ante.OracleKeeperI
type MockOracleKeeperI struct {
	votedMap map[string]bool
}

func (m MockOracleKeeperI) HasVotedInCurrentPeriod(ctx sdk.Context, valAddr sdk.ValAddress) bool {
	return m.votedMap[valAddr.String()]
}

// Test suite
type VoteFeeDiscountTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	app    *app.NibiruApp
	dec    ante.VoteFeeDiscountDecorator
	oracle *MockOracleKeeperI
	stake  *MockStakingKeeperI
}

// buildTestTx creates a transaction recognized by the standard SDK Tx decoder.
// - msgs: the messages the TX will carry
// - signers: the addresses that will appear as signers in `GetSigners()`
// NOTE: We create placeholder signatures for each signer so the TX won't fail
// any 'signer-length' checks internally.
func (s *VoteFeeDiscountTestSuite) buildTestTx(msgs []sdk.Msg, signers []sdk.AccAddress) (sdk.Tx, error) {
	// 1) Create a new TxBuilder from the app's TxConfig
	txBuilder := s.app.GetTxConfig().NewTxBuilder()

	// 2) Set the messages
	err := txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	// (Optional) Set any fees, memo, timeouts, etc. We can do zero fees here.
	txBuilder.SetFeeAmount(sdk.NewCoins()) // no fees
	txBuilder.SetGasLimit(0)               // gas=0 just for tests
	txBuilder.SetMemo("test-tx")
	// etc. as needed

	// 3) We add placeholder signatures for each signer
	//
	// The chain expects len(signatures) == len(signers). Even if the
	// signatures are empty or all zeros, we need them to match the number
	// of signers so that the Tx's `GetSigners()` aligns with what
	// the AnteHandler expects.

	sigs := make([]signing.SignatureV2, len(signers))
	for i := range signers {
		// We'll just produce a random private key each time, or you could
		// keep a single ephemeral key. The key isn't used in a real sig,
		// but the presence of the signature tells the SDK "this is a valid signer."
		privKey := ed25519.GenPrivKey()

		// Make an empty signature
		sigData := signing.SingleSignatureData{
			SignMode:  s.app.GetTxConfig().SignModeHandler().DefaultMode(),
			Signature: nil, // no actual signature bytes
		}
		sigV2 := signing.SignatureV2{
			PubKey:   privKey.PubKey(),
			Data:     &sigData,
			Sequence: 0,
		}
		sigs[i] = sigV2
	}

	err = txBuilder.SetSignatures(sigs...)
	if err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

func TestVoteFeeDiscountTestSuite(t *testing.T) {
	suite.Run(t, new(VoteFeeDiscountTestSuite))
}

func (s *VoteFeeDiscountTestSuite) SetupTest() {
	// Use a fresh Nibiru app to get a fresh context or make a BasicContext
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	s.app = nibiruApp
	s.ctx = ctx

	s.stake = &MockStakingKeeperI{
		validators:  map[string]types.Validator{},
		totalBonded: math.NewInt(100_000_000),
	}
	s.oracle = &MockOracleKeeperI{
		votedMap: map[string]bool{},
	}

	s.dec = ante.NewVoteFeeDiscountDecorator(s.oracle, s.stake)
}

func dummyNext(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ctx, nil
}

func (s *VoteFeeDiscountTestSuite) TestVoteFeeDiscount() {
	valAddr := sdk.ValAddress([]byte("validator----------------"))
	addressBytes, err := sdk.GetFromBech32(valAddr.String(), sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	if err != nil {
		panic(err)
	}
	// Convert the address bytes to an account address
	accAddress := sdk.AccAddress(addressBytes)

	// By default, no validator is found
	s.stake.validators[valAddr.String()] = types.Validator{
		OperatorAddress: valAddr.String(),
		Status:          types.Bonded,           // non-jailed
		Tokens:          math.NewInt(1_000_000), // enough tokens
		Jailed:          false,
	}

	s.Run("Happy path", func() {
		// We'll track the minGasPrice in the context. The discount is 1/69,420
		// if triggered. Let's see if it changes or not.
		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDecWithPrec(1, 0))} // "1stake"
		newCtx := s.ctx.WithMinGasPrices(mgp)
		// two signers
		msgs := []sdk.Msg{&oracletypes.MsgAggregateExchangeRatePrevote{
			Hash:      "hash",
			Feeder:    accAddress.String(),
			Validator: valAddr.String(),
		}}
		signers := []sdk.AccAddress{sdk.AccAddress("signer1")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		// Should pass directly to next. No discount.
		s.Require().NotEqual(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should be updated")
	})

	s.Run("Invalid Tx Type => returns error", func() {
		badTx := sdk.Tx(nil) // not SigVerifiableTx
		ctx, err := s.dec.AnteHandle(s.ctx, badTx, false, dummyNext)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), sdkerrors.ErrTxDecode.Error())
		s.Require().Equal(s.ctx, ctx) // original context
	})

	s.Run("Multiple signers => pass to next with no discount", func() {
		// We'll track the minGasPrice in the context. The discount is 1/69,420
		// if triggered. Let's see if it changes or not.
		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDecWithPrec(1, 0))} // "1stake"
		newCtx := s.ctx.WithMinGasPrices(mgp)
		// two signers
		msgs := []sdk.Msg{&oracletypes.MsgAggregateExchangeRatePrevote{
			Hash:      "hash",
			Feeder:    accAddress.String(),
			Validator: valAddr.String(),
		}}
		signers := []sdk.AccAddress{sdk.AccAddress("signer1"), sdk.AccAddress("signer2")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		// Should pass directly to next. No discount.
		s.Require().Equal(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should not be updated")
	})

	s.Run("Msg is not a prevote or vote => pass to next (no discount)", func() {
		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDec(3))} // 3stake
		newCtx := s.ctx.WithMinGasPrices(mgp)

		msgs := []sdk.Msg{&oracletypes.MsgAggregateExchangeRatePrevote{
			Hash:      "hash",
			Feeder:    accAddress.String(),
			Validator: valAddr.String(),
		}, &banktypes.MsgSend{FromAddress: "from", ToAddress: "to", Amount: sdk.NewCoins(sdk.NewInt64Coin("stake", 1))}}
		signers := []sdk.AccAddress{sdk.AccAddress("signer1")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		s.Require().Equal(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should not be updated")
	})

	// ----------------------
	s.Run("Validator not found => pass to next, no discount", func() {
		// Remove from stake validators so the address won't be found
		delete(s.stake.validators, valAddr.String())

		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDec(5))}
		newCtx := s.ctx.WithMinGasPrices(mgp)

		msgs := []sdk.Msg{
			&oracletypes.MsgAggregateExchangeRatePrevote{
				Hash:      "hash",
				Feeder:    accAddress.String(),
				Validator: valAddr.String(),
			},
		}

		signers := []sdk.AccAddress{sdk.AccAddress("signer1")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		s.Require().Equal(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should not be updated")

		// Put it back
		s.stake.validators[valAddr.String()] = types.Validator{
			OperatorAddress: valAddr.String(),
			Status:          types.Bonded,
			Tokens:          math.NewInt(1_000_000),
			Jailed:          false,
		}
	})

	s.Run("Validator is jailed => no discount", func() {
		// Mark jailed
		v := s.stake.validators[valAddr.String()]
		v.Jailed = true
		s.stake.validators[valAddr.String()] = v

		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDec(2))}
		newCtx := s.ctx.WithMinGasPrices(mgp)

		msgs := []sdk.Msg{
			&oracletypes.MsgAggregateExchangeRatePrevote{
				Hash:      "hash",
				Feeder:    accAddress.String(),
				Validator: valAddr.String(),
			},
		}

		signers := []sdk.AccAddress{sdk.AccAddress("signer1")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		s.Require().Equal(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should not be updated")

		// unjail for next tests
		v.Jailed = false
		s.stake.validators[valAddr.String()] = v
	})

	s.Run("Validator does not meet 0.5% threshold => no discount", func() {
		// Suppose totalBonded = 1_000_000_000,
		// We need at least 0.5% = 5_000_000 to qualify.
		// This validator has 1_000_000 => fails the threshold
		val := s.stake.validators[valAddr.String()]
		val.Tokens = math.NewInt(1_000)
		s.stake.validators[valAddr.String()] = val

		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDec(2))}
		newCtx := s.ctx.WithMinGasPrices(mgp)

		msgs := []sdk.Msg{
			&oracletypes.MsgAggregateExchangeRatePrevote{
				Hash:      "hash",
				Feeder:    accAddress.String(),
				Validator: valAddr.String(),
			},
		}

		signers := []sdk.AccAddress{sdk.AccAddress("signer1")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		s.Require().Equal(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should not be updated")

		// bump tokens so we can test discount in next subtest
		val.Tokens = math.NewInt(5_000_000)
		s.stake.validators[valAddr.String()] = val
	})

	s.Run("Validator meets threshold but has voted => no discount", func() {
		// Mark that they've voted
		s.oracle.votedMap[valAddr.String()] = true

		mgp := []sdk.DecCoin{sdk.NewDecCoinFromDec("stake", sdk.NewDecWithPrec(100, 1))} // 10 stake
		newCtx := s.ctx.WithMinGasPrices(mgp)

		msgs := []sdk.Msg{
			&oracletypes.MsgAggregateExchangeRatePrevote{
				Hash:      "hash",
				Feeder:    accAddress.String(),
				Validator: valAddr.String(),
			},
		}

		signers := []sdk.AccAddress{sdk.AccAddress("signer1")}
		tx, err := s.buildTestTx(msgs, signers)
		s.Require().NoError(err)

		ctx2, err := s.dec.AnteHandle(newCtx, tx, false, dummyNext)
		s.Require().NoError(err)
		// should remain the same
		s.Require().Equal(sdk.DecCoins(mgp), ctx2.MinGasPrices(), "should not be updated")
	})
}
