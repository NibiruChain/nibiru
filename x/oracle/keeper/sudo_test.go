package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestSuiteOracleExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(SuiteOracleSudo))
}

type SuiteOracleSudo struct {
	suite.Suite
}

func (s *SuiteOracleSudo) TestEditOracleParams() {
	nibiru, ctx := testapp.NewNibiruTestAppAndContext()

	// Change to all non-defaults to test EditOracleParams as a setter .
	votePeriod := sdk.NewInt(1_234)
	voteThreshold := sdk.MustNewDecFromStr("0.4")
	rewardBand := sdk.MustNewDecFromStr("0.5")
	whitelist := []string{"aave:usdc", "sol:usdc"}
	slashFraction := sdk.MustNewDecFromStr("0.5")
	slashWindow := sdk.NewInt(2)
	minValidPerWindow := sdk.MustNewDecFromStr("0.5")
	twapLookbackWindow := sdk.NewInt(int64(time.Second * 30))
	minVoters := sdk.NewInt(2)
	validatorFeeRatio := sdk.MustNewDecFromStr("0.7")
	partialParams := oracletypes.MsgEditOracleParams{
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
	// TODO: Verify that params before were not equal

	invalidSender := testutil.AccAddress()
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(nibiru.OracleKeeper)
	goCtx := sdk.WrapSDKContext(ctx)
	partialParams.Sender = invalidSender.String()
	_, err := oracleMsgServer.EditOracleParams(
		goCtx, &partialParams,
	)
	s.Error(err)

	okSender := testapp.DefaultSudoRoot()
	partialParams.Sender = okSender.String()
	_, err = oracleMsgServer.EditOracleParams(
		goCtx, &partialParams,
	)
	s.NoError(err)

	// call admin method without err
}
