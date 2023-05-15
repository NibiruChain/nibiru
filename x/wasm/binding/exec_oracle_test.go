package binding_test

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"testing"
	"time"
)

func TestSuiteOracleExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteOracleExecutor))
}

type TestSuiteOracleExecutor struct {
	suite.Suite

	nibiru           *app.NibiruApp
	contractDeployer sdk.AccAddress
}

func (s *TestSuiteOracleExecutor) SetupSuite() {
	sender := testutil.AccAddress()
	s.contractDeployer = sender

	genesisState := SetupPerpGenesis()
	nibiru := testapp.NewNibiruTestApp(genesisState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})

	coins := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)),
		sdk.NewCoin(denoms.NUSD, sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))
}
