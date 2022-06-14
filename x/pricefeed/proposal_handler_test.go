package pricefeed_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/pricefeed"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestHandleWhitelistPriceOracleProposal(t *testing.T) {
	oracleAddress := sample.AccAddress()

	nibiruApp, ctx := testutil.NewNibiruApp(true)

	authorizedAddresses := nibiruApp.PricefeedKeeper.GetAuthorizedAddresses(ctx)
	require.NotContains(t, authorizedAddresses, oracleAddress)

	proposal := types.NewWhitelistPriceOracleProposal(
		"whitelist oracle",
		"the oracle with addr",
		oracleAddress,
	)

	handler := pricefeed.NewPriceFeedProposalHandler(nibiruApp.PricefeedKeeper)
	err := handler(ctx, proposal)
	require.NoError(t, err)

	updatedAuthorizedAddresses := nibiruApp.PricefeedKeeper.GetAuthorizedAddresses(ctx)
	require.Contains(t, updatedAuthorizedAddresses, oracleAddress)
}
