package testutil_test

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/common/testutil"
)

type TestSuite struct {
	suite.Suite
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestGetPackageDir() {
	pkgDir, err := testutil.GetPackageDir()
	s.NoError(err)
	s.Equal("testutil", path.Base(pkgDir))
	s.Equal("common", path.Base(path.Dir(pkgDir)))
}

// TestSampleFns: Tests functions that generate test data from sample.go
func (s *TestSuite) TestSampleFns() {
	s.T().Log("consecutive calls give different addrs")
	addrs := set.New[string]()
	for times := 0; times < 16; times++ {
		newAddr := testutil.AccAddress().String()
		s.False(addrs.Has(newAddr))
		addrs.Add(newAddr)
	}
}

func (s *TestSuite) TestPrivKeyAddressPairs() {
	s.T().Log("calls should be deterministic")
	keysA, addrsA := testutil.PrivKeyAddressPairs(4)
	keysB, addrsB := testutil.PrivKeyAddressPairs(4)
	s.Equal(keysA, keysB)
	s.Equal(addrsA, addrsB)
}

func (s *TestSuite) TestBlankContext() {
	ctx := testutil.BlankContext("new-kv-store-key")
	goCtx := sdk.WrapSDKContext(ctx)

	freshGoCtx := context.Background()
	s.Require().Panics(func() { sdk.UnwrapSDKContext(freshGoCtx) })

	s.Require().NotPanics(func() { sdk.UnwrapSDKContext(goCtx) })
}
