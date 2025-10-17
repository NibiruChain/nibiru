package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/bank/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
)

// ------------------------------------------------------------
// Test suite struct definitions

type NibiruBankSuite struct {
	testutil.LogRoutingSuite
}

// TestNibiruBank: Runs all the tests in the suite.
func TestNibiruBank(t *testing.T) {
	suite.Run(t, new(NibiruBankSuite))
}

var _ suite.SetupAllSuite = (*NibiruBankSuite)(nil)

// ------------------------------------------------------------
// Tests

func (s *NibiruBankSuite) TestAddWei() {
	expectWeiChangeEvents := func(
		deps evmtest.TestDeps,
		numAdd int,
		changedAcc sdk.AccAddress,
	) {
		addEvents := testutil.FindEventsOfType(
			deps.Ctx().EventManager().Events(),
			bank.EventTypeWeiChange,
		)
		s.Len(addEvents, numAdd, "expect wei change events")

		if len(addEvents) > 0 {
			attrReason := addEvents[0].Attributes[0]
			wantAttrReason := bank.WeiChangeReason_AddWei
			s.Equal(wantAttrReason.GetKey(), attrReason.GetKey())
			s.Equal(wantAttrReason.GetValue(), attrReason.GetValue())

			attrAddrs := addEvents[0].Attributes[1]
			s.Equal(bank.AttributeKeyWeiChangeAddrs, attrAddrs.GetKey())
			s.Contains(changedAcc.String(), attrAddrs.GetValue(), "expect changed account in events")
		}
	}
	wei := func(n uint64) *uint256.Int { return uint256.NewInt(n) }
	var (
		acc  = evmtest.NewEthPrivAcc()
		addr = acc.NibiruAddr
	)
	testCases := testutil.FunctionTestCases{
		{
			Name: "No-op for AddWei(nil | zero)",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper
				s.Equal(
					uint256.NewInt(0),
					k.GetWeiBalance(deps.Ctx(), addr),
				)

				k.AddWei(deps.Ctx(), addr, wei(0))
				expectWeiChangeEvents(deps, 0, addr)

				k.AddWei(deps.Ctx(), addr, wei(500)) // emits event
				k.AddWei(deps.Ctx(), addr, wei(0))
				expectWeiChangeEvents(deps, 1, addr)
				s.Equal(
					wei(500),
					k.GetWeiBalance(deps.Ctx(), addr),
				)
				s.Equal(
					wei(500).String(),
					k.WeiBlockDelta(deps.Ctx()).String(),
				)
			},
		},
		{
			Name: "Below threshold (< WeiPerUnibi) updates only wei store",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				WeiPerUnibiMinus1 := new(uint256.Int).
					Sub(nutil.WeiPerUnibiU256(), wei(1))
				k.AddWei(deps.Ctx(), addr, WeiPerUnibiMinus1)
				s.Equal(
					"0",
					k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String(),
					"below micronibi threshold, expect no bank coin change",
				)
				s.Equal(
					WeiPerUnibiMinus1,
					k.GetWeiBalance(deps.Ctx(), addr),
				)
				expectWeiChangeEvents(deps, 1, addr)
				s.Equal(
					WeiPerUnibiMinus1.String(),
					k.WeiBlockDelta(deps.Ctx()).String(),
				)
			},
		},
		{
			Name: "Crossing unibi threshold (WeiPerUnibi) changes bank coin balance",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				sum := wei(0)

				doAdd := func(nextAdd *uint256.Int) {
					sum = new(uint256.Int).Add(sum, nextAdd)
					k.AddWei(deps.Ctx(), addr, nextAdd)
				}

				for _, u := range []*uint256.Int{
					wei(420),                     // += 420 attonibi
					evm.NativeToWeiU256(wei(69)), // += 69*10^{12} attonibi
					wei(5_500_000_000_000),       // += 5.5*10^{12} attonibi
				} {
					doAdd(u)
				}

				s.Equal(
					"74", // == 69 + 5 micronibi
					k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String(),
					"expect 69 + 5 micronibi",
				)
				s.Equal(
					wei(74_500_000_000_420),
					k.GetWeiBalance(deps.Ctx(), addr),
					"expect true balance be correct",
				)
				s.Equal(
					sum,
					k.GetWeiBalance(deps.Ctx(), addr),
					"expect true balance to match sum",
				)
				expectWeiChangeEvents(deps, 3, addr)
			},
		},
	}
	testutil.RunFunctionTestSuite(&s.Suite, testCases)
}

func (s *NibiruBankSuite) TestSubWei() {
	wei := func(n uint64) *uint256.Int { return uint256.NewInt(n) }
	expectWeiChangeEvents := func(
		deps evmtest.TestDeps,
		num int,
		reason sdk.Attribute,
		containsAddr string,
	) {
		evs := testutil.FindEventsOfType(deps.Ctx().EventManager().Events(), bank.EventTypeWeiChange)
		s.Len(evs, num)
		if num > 0 {
			attrReason := evs[num-1].Attributes[0]
			s.Equal(reason.GetKey(), attrReason.GetKey())
			s.Equal(reason.GetValue(), attrReason.GetValue())
			if containsAddr != "" {
				attrAddrs := evs[num-1].Attributes[1]
				s.Equal(bank.AttributeKeyWeiChangeAddrs, attrAddrs.GetKey())
				s.Contains(attrAddrs.GetValue(), containsAddr)
			}
		}
	}

	var (
		acc  = evmtest.NewEthPrivAcc()
		addr = acc.NibiruAddr
	)

	testCases := testutil.FunctionTestCases{
		{
			Name: "No-op for SubWei(nil | zero)",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				// Start with some wei to make sure SubWei(0) doesn't change anything
				k.AddWei(deps.Ctx(), addr, wei(777))
				preBal := k.GetWeiBalance(deps.Ctx(), addr).Clone()

				s.NoError(k.SubWei(deps.Ctx(), addr, wei(0)))
				s.Equal(
					preBal.String(),
					k.GetWeiBalance(deps.Ctx(), addr).String(),
				)
				// Only the prior AddWei emitted one event
				expectWeiChangeEvents(deps, 1, bank.WeiChangeReason_AddWei, addr.String())
			},
		},
		{
			Name: "From wei-store only (wei-store >= amt): deducts store; unibi unchanged",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				k.AddWei(deps.Ctx(), addr, wei(10_000)) // below threshold, goes only to store
				preUnibi := k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String()

				s.NoError(k.SubWei(deps.Ctx(), addr, wei(3_000)))
				s.Equal(
					preUnibi,
					k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String(),
				)
				s.Equal(
					"7000",
					k.GetWeiBalance(deps.Ctx(), addr).String(),
				)
				// delta should be +10000 - 3000 = 7000
				s.Equal(
					"7000",
					k.WeiBlockDelta(deps.Ctx()).String(),
				)
				expectWeiChangeEvents(deps, 2, bank.WeiChangeReason_SubWei, addr.String())
			},
		},
		{
			Name: "Pull from unibi when wei-store < amt, aggregate sufficient, with remainder",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				// Build a balance: 2 unibi + 500 wei
				k.AddWei(deps.Ctx(), addr, evm.NativeToWeiU256(wei(2))) // mints 2 unibi
				k.AddWei(deps.Ctx(), addr, wei(500))
				s.Equal("2", k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String())
				s.Equal(uint256.NewInt(0).Add(evm.NativeToWeiU256(wei(2)), wei(500)).String(), k.GetWeiBalance(deps.Ctx(), addr).String())

				// Sub 1 unibi + 200 wei = 1*1e12 + 200
				subAmt := new(uint256.Int).Add(evm.NativeToWeiU256(wei(1)), wei(200))
				s.NoError(k.SubWei(deps.Ctx(), addr, subAmt))

				// Expect new unibi = 1; wei-store = 300 (since 500 - 200 after 1 unibi borrowed)
				s.Equal(
					"1",
					k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String(),
				)
				s.Equal(
					new(uint256.Int).Add(evm.NativeToWeiU256(wei(1)), wei(300)).String(),
					k.GetWeiBalance(deps.Ctx(), addr).String(),
				)

				// delta = +2e12 + 500 - (1e12 + 200) = +1e12 + 300
				s.Equal(
					new(uint256.Int).Add(evm.NativeToWeiU256(wei(1)), wei(300)).String(),
					k.WeiBlockDelta(deps.Ctx()).String(),
				)
				expectWeiChangeEvents(deps, 3, bank.WeiChangeReason_SubWei, addr.String())
			},
		},
		{
			Name: "Edge: subtract exactly aggregate -> both unibi and wei-store zero",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				// 3 unibi + 999 wei
				k.AddWei(deps.Ctx(), addr, evm.NativeToWeiU256(wei(3)))
				k.AddWei(deps.Ctx(), addr, wei(999))

				total := k.GetWeiBalance(deps.Ctx(), addr).Clone()
				s.NoError(k.SubWei(deps.Ctx(), addr, total))

				s.Equal("0", k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String())
				s.Equal("0", k.GetWeiBalance(deps.Ctx(), addr).String())
				// delta should be zero (adds then total sub)
				s.Equal("0", k.WeiBlockDelta(deps.Ctx()).String())
				expectWeiChangeEvents(deps, 3, bank.WeiChangeReason_SubWei, addr.String())
			},
		},
		{
			Name: "Edge: subtract leaving only wei-store residue",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				// 5 unibi + 1 wei
				k.AddWei(deps.Ctx(), addr, evm.NativeToWeiU256(wei(5)))
				k.AddWei(deps.Ctx(), addr, wei(1))

				// Sub 4 unibi + 999,999,999,999 wei -> leaves 2 wei residue
				subAmt := new(uint256.Int).Add(evm.NativeToWeiU256(wei(4)), uint256.NewInt(999_999_999_999))
				s.NoError(k.SubWei(deps.Ctx(), addr, subAmt))

				// 5*1e12+1 - (4*1e12 + 999,999,999,999) = 2
				s.Equal("0", k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String())
				s.Equal("2", k.GetWeiBalance(deps.Ctx(), addr).String())
				expectWeiChangeEvents(deps, 3, bank.WeiChangeReason_SubWei, addr.String())
			},
		},
		{
			Name: "Insufficient funds: error; no state or delta mutation",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				// Start with 1 unibi + 0 wei
				k.AddWei(deps.Ctx(), addr, evm.NativeToWeiU256(wei(1)))
				preWei := k.GetWeiBalance(deps.Ctx(), addr).Clone()
				preUnibi := k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String()
				preDelta := k.WeiBlockDelta(deps.Ctx()).String()

				// Try to subtract more than aggregate (1 unibi + 1 wei)
				err := k.SubWei(deps.Ctx(), addr, new(uint256.Int).Add(evm.NativeToWeiU256(wei(1)), wei(1)))
				s.Error(err)

				// No change on balances or delta
				s.Equal(preWei.String(), k.GetWeiBalance(deps.Ctx(), addr).String())
				s.Equal(preUnibi, k.GetBalance(deps.Ctx(), addr, appconst.DENOM_UNIBI).Amount.String())
				s.Equal(preDelta, k.WeiBlockDelta(deps.Ctx()).String())

				// Only the AddWei emitted event; no SubWei event on error
				expectWeiChangeEvents(deps, 1, bank.WeiChangeReason_AddWei, addr.String())
			},
		},
		{
			Name: "WeiBlockDelta decreases by subtracted wei",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				k.AddWei(deps.Ctx(), addr, wei(1_000))
				k.AddWei(deps.Ctx(), addr, evm.NativeToWeiU256(wei(1))) // +1e12

				// delta currently 1e12 + 1000
				s.Equal(new(uint256.Int).Add(evm.NativeToWeiU256(wei(1)), wei(1_000)).String(), k.WeiBlockDelta(deps.Ctx()).String())

				s.NoError(k.SubWei(deps.Ctx(), addr, wei(250)))
				// delta becomes (1e12 + 1000 - 250)
				s.Equal(new(uint256.Int).Add(evm.NativeToWeiU256(wei(1)), wei(750)).String(), k.WeiBlockDelta(deps.Ctx()).String())
			},
		},
		{
			Name: "Add then Sub same amount (below/over threshold) restores original state",
			Test: func() {
				deps := evmtest.NewTestDeps()
				k := deps.App.BankKeeper

				type op struct{ add, sub *uint256.Int }
				ops := []op{
					{add: wei(999_999_999_999), sub: wei(999_999_999_999)},               // below threshold
					{add: evm.NativeToWeiU256(wei(3)), sub: evm.NativeToWeiU256(wei(3))}, // exact multiples
					{add: new(uint256.Int).Add(evm.NativeToWeiU256(wei(2)), wei(123)), sub: new(uint256.Int).Add(evm.NativeToWeiU256(wei(2)), wei(123))},
				}

				orig := k.GetWeiBalance(deps.Ctx(), addr).Clone()
				for _, o := range ops {
					k.AddWei(deps.Ctx(), addr, o.add)
					s.NoError(k.SubWei(deps.Ctx(), addr, o.sub))
				}

				s.Equal(orig.String(), k.GetWeiBalance(deps.Ctx(), addr).String())
				// Net delta should be unchanged as well
				s.Equal("0", k.WeiBlockDelta(deps.Ctx()).String())
			},
		},
	}

	testutil.RunFunctionTestSuite(&s.Suite, testCases)
}

func (s *NibiruBankSuite) TestSumWeiStoreBals() {
	wei := func(n uint64) *uint256.Int { return uint256.NewInt(n) }

	deps := evmtest.NewTestDeps()
	k := deps.App.BankKeeper

	// Three fresh accounts
	accA := evmtest.NewEthPrivAcc()
	accB := evmtest.NewEthPrivAcc()
	accC := evmtest.NewEthPrivAcc()

	// Account A: cross the 10^12 threshold so that it mints 1 unibi
	// and leaves a small wei-store remainder of 7.
	k.AddWei(deps.Ctx(), accA.NibiruAddr, evm.NativeToWeiU256(wei(1))) // +1e12 -> 1 unibi
	k.AddWei(deps.Ctx(), accA.NibiruAddr, wei(7))                      // remainder -> wei-store = 7

	// Accounts B and C: simple store-only balances
	k.AddWei(deps.Ctx(), accB.NibiruAddr, wei(420))
	k.AddWei(deps.Ctx(), accC.NibiruAddr, wei(69))

	// Sum should include only wei-store remainders: 7 + 420 + 69 = 496
	sum := k.SumWeiStoreBals(deps.Ctx())
	s.Equal("496", sum.String())
}

// TestTotalSupplyInvariant tests the TotalSupply invariant with both NIBI (dual-balance model)
// and non-NIBI coins. It verifies that NIBI mismatches are allowed due to wei-store
// but non-NIBI coin mismatches break the invariant.
func (s *NibiruBankSuite) TestTotalSupplyInvariant() {
	wei := func(n uint64) *uint256.Int { return uint256.NewInt(n) }

	deps := evmtest.NewTestDeps()
	k := deps.App.BankKeeper

	s.T().Log("Creating two test accounts for TotalSupply invariant test")
	accA := evmtest.NewEthPrivAcc()
	accB := evmtest.NewEthPrivAcc()
	accC := evmtest.NewEthPrivAcc()

	s.T().Log("Test Case 1: Happy Path - NIBI with wei-store should pass invariant")
	s.T().Log("Adding NIBI balances that cross the 10^{12} threshold to create unibi + wei-store")
	k.AddWei(deps.Ctx(), accA.NibiruAddr, evm.NativeToWeiU256(wei(2))) // 2×10^{12} wei -> 2 unibi
	k.AddWei(deps.Ctx(), accA.NibiruAddr, wei(500))                    // 500 wei remainder
	k.AddWei(deps.Ctx(), accB.NibiruAddr, evm.NativeToWeiU256(wei(1))) // 1×10^{12} wei -> 1 unibi
	k.AddWei(deps.Ctx(), accB.NibiruAddr, wei(300))                    // 300 wei remainder

	coinsTest := sdk.NewCoins(sdk.NewInt64Coin("testcoin", 1000))
	coinsUnibi := sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 69))
	for _, coins := range []sdk.Coins{
		coinsTest, coinsUnibi,
	} {
		err := testapp.FundAccount(k, deps.Ctx(), accC.NibiruAddr, coins)
		s.NoErrorf(err, "should fund account A with %s successfully", coins)
	}

	s.T().Log("Calling TotalSupply invariant - should pass (NIBI mismatches allowed due to wei-store)")
	invarMsg, broken := keeper.TotalSupply(k)(deps.Ctx())
	s.False(broken, "NIBI with wei-store should pass invariant: %s", invarMsg)

	// TODO: test -> Create mismatch on purpose by burning to create sad path
	// test case. Goal:
	// s.T().Log("Creating mismatch by burning coins from supply without updating account balance")
	// s.T().Log("This creates: sum(account balances) > TotalSupply for testcoin")
}
