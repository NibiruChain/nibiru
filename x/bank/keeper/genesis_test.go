package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/holiman/uint256"
)

func (suite *KeeperTestSuite) TestExportGenesis() {
	ctx := suite.ctx

	expectedMetadata := suite.getTestMetadata()
	expectedBalances, expTotalSupply := suite.getTestBalancesAndSupply()

	// Adding genesis supply to the expTotalSupply
	genesisSupply, _, err := suite.bankKeeper.GetPaginatedTotalSupply(suite.ctx, &query.PageRequest{Limit: query.MaxLimit})
	suite.Require().NoError(err)
	expTotalSupply = expTotalSupply.Add(genesisSupply...)

	for i := range []int{1, 2} {
		suite.bankKeeper.SetDenomMetaData(ctx, expectedMetadata[i])
		accAddr, err1 := sdk.AccAddressFromBech32(expectedBalances[i].Address)
		if err1 != nil {
			panic(err1)
		}
		// set balances via mint and send
		suite.mockMintCoins(mintAcc)
		suite.
			Require().
			NoError(suite.bankKeeper.MintCoins(ctx, minttypes.ModuleName, expectedBalances[i].Coins))
		suite.mockSendCoinsFromModuleToAccount(mintAcc, accAddr)
		suite.
			Require().
			NoError(suite.bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, accAddr, expectedBalances[i].Coins))
	}

	suite.Require().NoError(suite.bankKeeper.SetParams(ctx, types.DefaultParams()))

	// Seed a couple of wei-store balances then export
	a0, _ := sdk.AccAddressFromBech32("nibi16xk72qeyy59px3cu63c2frl6vw5h2q6rg635z0")
	a1, _ := sdk.AccAddressFromBech32("nibi1zv4n28t9f9tadlha9umlfdppnh95qd65uge4gt")
	// Use AddWei to set wei-store small values without crossing threshold
	suite.bankKeeper.AddWei(ctx, a0, uint256.NewInt(7))
	suite.bankKeeper.AddWei(ctx, a1, uint256.NewInt(420))

	exportGenesis := suite.bankKeeper.ExportGenesis(ctx)

	suite.Require().Len(exportGenesis.Params.SendEnabled, 0)
	suite.Require().Equal(types.DefaultParams().DefaultSendEnabled, exportGenesis.Params.DefaultSendEnabled)
	suite.Require().Equal(expTotalSupply, exportGenesis.Supply)
	suite.Require().Subset(exportGenesis.Balances, expectedBalances)
	suite.Require().Equal(expectedMetadata, exportGenesis.DenomMetadata)

	// Assert wei_balances exported and contain our entries
	// Order should be ascending by address; just verify presence and values
	found := map[string]string{}
	for _, wb := range exportGenesis.WeiBalances {
		found[wb.AddrBech32] = fmt.Sprintf("%d", wb.WeiStoreBal)
	}
	suite.Require().Equal("7", found[a0.String()])
	suite.Require().Equal("420", found[a1.String()])
}

func (suite *KeeperTestSuite) getTestBalancesAndSupply() ([]types.Balance, sdk.Coins) {
	addr2, _ := sdk.AccAddressFromBech32("nibi124hwhl9vjk8g659cy9220qdwaw0asan50xzwyx")
	addr1, _ := sdk.AccAddressFromBech32("nibi1jxsk9yla447rs98sccjfcng4d3ax6ptnxhjmpu")
	addr1Balance := sdk.Coins{sdk.NewInt64Coin("testcoin3", 10)}
	addr2Balance := sdk.Coins{sdk.NewInt64Coin("testcoin1", 32), sdk.NewInt64Coin("testcoin2", 34)}

	totalSupply := addr1Balance
	totalSupply = totalSupply.Add(addr2Balance...)

	return []types.Balance{
		{Address: addr2.String(), Coins: addr2Balance},
		{Address: addr1.String(), Coins: addr1Balance},
	}, totalSupply
}

func (suite *KeeperTestSuite) TestInitGenesis() {
	m := types.Metadata{Description: sdk.DefaultBondDenom, Base: sdk.DefaultBondDenom, Display: sdk.DefaultBondDenom}
	g := types.DefaultGenesisState()
	g.DenomMetadata = []types.Metadata{m}
	// Add wei_balances entries (< 1e12)
	g.WeiBalances = []types.WeiBalance{
		{AddrBech32: "nibi16xk72qeyy59px3cu63c2frl6vw5h2q6rg635z0", WeiStoreBal: 7},
		{AddrBech32: "nibi1zv4n28t9f9tadlha9umlfdppnh95qd65uge4gt", WeiStoreBal: 420},
	}

	bk := suite.bankKeeper
	bk.InitGenesis(suite.ctx, g)

	m2, found := bk.GetDenomMetaData(suite.ctx, m.Base)
	suite.Require().True(found)
	suite.Require().Equal(m, m2)

	// Validate wei store balances initialized
	addr0, _ := sdk.AccAddressFromBech32(g.WeiBalances[0].AddrBech32)
	addr1, _ := sdk.AccAddressFromBech32(g.WeiBalances[1].AddrBech32)
	suite.Require().Equal("7", bk.GetWeiBalance(suite.ctx, addr0).String())
	suite.Require().Equal("420", bk.GetWeiBalance(suite.ctx, addr1).String())
}

func (suite *KeeperTestSuite) TestTotalSupply() {
	// Prepare some test data.
	defaultGenesis := types.DefaultGenesisState()

	addrs := []string{
		"nibi16xk72qeyy59px3cu63c2frl6vw5h2q6rg635z0",
		"nibi1zv4n28t9f9tadlha9umlfdppnh95qd65uge4gt",
		"nibi1jc42gyjugezx7s2drl6vgrgwd34skcgelgdnr2",
	}

	balances := []types.Balance{
		{Coins: sdk.NewCoins(sdk.NewCoin("foocoin", sdk.NewInt(1))), Address: addrs[0]},
		{Coins: sdk.NewCoins(sdk.NewCoin("barcoin", sdk.NewInt(1))), Address: addrs[1]},
		{Coins: sdk.NewCoins(sdk.NewCoin("foocoin", sdk.NewInt(10)), sdk.NewCoin("barcoin", sdk.NewInt(20))), Address: addrs[2]},
	}
	totalSupply := sdk.NewCoins(sdk.NewCoin("foocoin", sdk.NewInt(11)), sdk.NewCoin("barcoin", sdk.NewInt(21)))

	genesisSupply, _, err := suite.bankKeeper.GetPaginatedTotalSupply(suite.ctx, &query.PageRequest{Limit: query.MaxLimit})
	suite.Require().NoError(err)

	testcases := []struct {
		name        string
		genesis     *types.GenesisState
		expSupply   sdk.Coins
		expPanic    bool
		expPanicMsg string
	}{
		{
			"calculation NOT matching genesis Supply field",
			types.NewGenesisState(defaultGenesis.Params, balances, sdk.NewCoins(sdk.NewCoin("wrongcoin", sdk.NewInt(1))), defaultGenesis.DenomMetadata, defaultGenesis.SendEnabled),
			nil, true, "genesis supply is incorrect, expected 1wrongcoin, got 21barcoin,11foocoin",
		},
		{
			"calculation matches genesis Supply field",
			types.NewGenesisState(defaultGenesis.Params, balances, totalSupply, defaultGenesis.DenomMetadata, defaultGenesis.SendEnabled),
			totalSupply, false, "",
		},
		{
			"calculation is correct, empty genesis Supply field",
			types.NewGenesisState(defaultGenesis.Params, balances, nil, defaultGenesis.DenomMetadata, defaultGenesis.SendEnabled),
			totalSupply, false, "",
		},
	}

	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			if tc.expPanic {
				suite.PanicsWithError(tc.expPanicMsg, func() { suite.bankKeeper.InitGenesis(suite.ctx, tc.genesis) })
			} else {
				suite.bankKeeper.InitGenesis(suite.ctx, tc.genesis)
				totalSupply, _, err := suite.bankKeeper.GetPaginatedTotalSupply(suite.ctx, &query.PageRequest{Limit: query.MaxLimit})
				suite.Require().NoError(err)

				// adding genesis supply to expected supply
				expected := tc.expSupply.Add(genesisSupply...)
				suite.Require().Equal(expected, totalSupply)
			}
		})
	}
}
