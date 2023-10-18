package testutil_test

import (
	"context"
	"os/exec"
	"path"
	"testing"

	"github.com/spf13/cobra"
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

func (s *TestSuite) TestNullifyFill() {
	for _, tc := range []struct {
		name  string
		input any
		want  any
	}{
		{
			name:  "nullify fill slice",
			input: []string{},
			want:  make([]string, 0),
		},
		{
			name: "nullify fill struct with coins",
			input: struct {
				Coins   sdk.Coins
				Strings []string
			}{},
			want: struct {
				Coins   sdk.Coins
				Strings []string
			}{
				Coins:   sdk.Coins(nil),
				Strings: []string(nil),
			},
		},
		{
			name: "nullify fill sdk.Coin struct",
			input: struct {
				Coin sdk.Coin
				Ints []int
			}{},
			want: struct {
				Coin sdk.Coin
				Ints []int
			}{
				Coin: sdk.Coin{},
				Ints: []int(nil),
			},
		},
		{
			name:  "nullify fill pointer to null concrete",
			input: new(sdk.Coin),
			want:  sdk.Coin{},
		},
	} {
		s.Run(tc.name, func() {
			got := testutil.Fill(tc.input)
			s.EqualValues(tc.want, got)
		})
	}
}

func (s *TestSuite) TestSetupClientCtx() {
	goCtx := testutil.SetupClientCtx(s.T())
	trivialCobraCommand := &cobra.Command{
		Use:   "run-true",
		Short: "Runs the Unix command, 'true'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return exec.Command("true").Run()
		},
	}

	err := trivialCobraCommand.ExecuteContext(goCtx)
	s.NoError(err)
}
