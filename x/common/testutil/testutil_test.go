package testutil_test

import (
	"path"
	"testing"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/stretchr/testify/suite"
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
