package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

type KeeperSuite struct {
	suite.Suite
}

// TestKeeperSuite: Runs all of the tests in the suite.
func TestKeeperSuite(t *testing.T) {
	s := new(KeeperSuite)
	suite.Run(t, s)
}

func (s *KeeperSuite) TestQuerier() {
	chain, ctx := testapp.NewNibiruTestAppAndContext()
	goCtx := sdk.WrapSDKContext(ctx)
	k := chain.EvmKeeper
	for _, testCase := range []func() error{
		func() error {
			_, err := k.BaseFee(goCtx, nil)
			return err
		},
		func() error {
			_, err := k.EthCall(goCtx, nil)
			return err
		},
		func() error {
			_, err := k.EstimateGas(goCtx, nil)
			return err
		},
		func() error {
			_, err := k.TraceTx(goCtx, nil)
			return err
		},
		func() error {
			_, err := k.TraceBlock(goCtx, nil)
			return err
		},
	} {
		err := testCase()
		s.Require().ErrorContains(err, common.ErrNotImplemented().Error())
	}
}
