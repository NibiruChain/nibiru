package keeper_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/testutil"

	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func (s *TestSuite) TestStoreWrite() {
	numCreators := 2
	subdenoms := []string{"aaaa", "bbbb"}
	tfdenoms := []tftypes.TFDenom{}
	for idx := 0; idx < numCreators; idx++ {
		_, creator := testutil.PrivKey()
		for _, subdenom := range subdenoms {
			tfdenom := tftypes.TFDenom{
				Creator:  creator.String(),
				Subdenom: subdenom,
			}
			tfdenoms = append(tfdenoms, tfdenom)
		}
	}

	api := s.keeper.Store

	s.T().Run("initial conditions", func(t *testing.T) {
		for _, tfdenom := range tfdenoms {
			// created denoms should be valid
			fmt.Printf("tfdenom: %v\n", tfdenom)
			s.NoError(tfdenom.Validate(), tfdenom)

			// query by denom should fail for all denoms
			_, err := api.Denoms.Get(s.ctx, tfdenom.String())
			s.Error(err)

			// query by creator should fail for all addrs
			s.False(api.HasCreator(s.ctx, tfdenom.Creator))
		}
	})

	s.T().Run("insert to state", func(t *testing.T) {
		// inserting should succeed
		for _, tfdenom := range tfdenoms {
			api.MustInsertDenom(s.ctx, tfdenom)
		}

		allDenoms := api.Denoms.Iterate(
			s.ctx, collections.Range[string]{}).Values()
		s.Len(allDenoms, numCreators*len(subdenoms))

		for _, tfdenom := range tfdenoms {
			s.True(api.HasCreator(s.ctx, tfdenom.Creator))
		}

		// query by creator should fail for a random addr
		s.False(api.HasCreator(s.ctx, testutil.AccAddress().String()))
	})

	s.T().Run("inserting invalid denom should fail", func(t *testing.T) {
		blankDenom := tftypes.TFDenom{}
		s.Error(blankDenom.Validate())
		s.Error(api.InsertDenom(s.ctx, blankDenom))
	})
}
