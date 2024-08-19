package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

// TestSuiteOracleSudo tests sudo-only functions in the oracle module.
func TestSuiteOracleSudo(t *testing.T) {
	suite.Run(t, new(SuiteOracleSudo))
}

type SuiteOracleSudo struct {
	suite.Suite
}

// TestEditOracleParams tests the business logic for
// "oraclekeeper.Keeper.Sudo().EditOracleParams"
func (s *SuiteOracleSudo) TestEditOracleParams() {
	nibiru, ctx := testapp.NewNibiruTestAppAndContext()

	// Change to all non-defaults to test EditOracleParams as a setter .
	votePeriod := math.NewInt(1_234)
	voteThreshold := math.LegacyMustNewDecFromStr("0.4")
	rewardBand := math.LegacyMustNewDecFromStr("0.5")
	whitelist := []string{"aave:usdc", "sol:usdc"}
	slashFraction := math.LegacyMustNewDecFromStr("0.5")
	slashWindow := math.NewInt(2_000)
	minValidPerWindow := math.LegacyMustNewDecFromStr("0.5")
	twapLookbackWindow := math.NewInt(int64(time.Second * 30))
	minVoters := math.NewInt(2)
	validatorFeeRatio := math.LegacyMustNewDecFromStr("0.7")
	msgEditParams := oracletypes.MsgEditOracleParams{
		VotePeriod:         &votePeriod,
		VoteThreshold:      &voteThreshold,
		RewardBand:         &rewardBand,
		Whitelist:          whitelist,
		SlashFraction:      &slashFraction,
		SlashWindow:        &slashWindow,
		MinValidPerWindow:  &minValidPerWindow,
		TwapLookbackWindow: &twapLookbackWindow,
		MinVoters:          &minVoters,
		ValidatorFeeRatio:  &validatorFeeRatio,
	}

	s.T().Log("Params before MUST NOT be equal to default")
	defaultParams := oracletypes.DefaultParams()
	currParams, err := nibiru.OracleKeeper.Params.Get(ctx)
	s.NoError(err)
	s.Equal(currParams, defaultParams,
		"Current params should be eqaul to defaults")
	partialParams := msgEditParams
	fullParams := oraclekeeper.MergeOracleParams(partialParams, defaultParams)
	s.NotEqual(defaultParams, fullParams,
		"new params after merge should not be defaults")

	invalidSender := testutil.AccAddress()
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(nibiru.OracleKeeper)
	goCtx := sdk.WrapSDKContext(ctx)
	msgEditParams.Sender = invalidSender.String()
	_, err = oracleMsgServer.EditOracleParams(
		goCtx, &msgEditParams,
	)
	s.Error(err)

	s.T().Log("Params after MUST be equal to new ones with partialParams")
	okSender := testapp.DefaultSudoRoot()
	msgEditParams.Sender = okSender.String()
	resp, err := oracleMsgServer.EditOracleParams(
		goCtx, &msgEditParams,
	)
	s.Require().NoError(err)
	s.EqualValues(resp.NewParams.String(), fullParams.String())

	s.T().Log("Changing to invalid params MUST fail")
	slashWindow = math.NewInt(1_233) // slashWindow < vote period is not allowed.
	msgEditParams = oracletypes.MsgEditOracleParams{
		Sender:      okSender.String(),
		SlashWindow: &slashWindow,
	}
	_, err = oracleMsgServer.EditOracleParams(
		goCtx, &msgEditParams,
	)
	s.Require().Error(err)
	s.ErrorContains(err, "oracle parameter SlashWindow must be greater")
}
