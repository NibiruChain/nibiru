package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MsgsTestSuite struct {
	suite.Suite
	contract      sdk.AccAddress
	deployer      sdk.AccAddress
	deployerStr   string
	withdrawerStr string
}

func TestMsgsTestSuite(t *testing.T) {
	suite.Run(t, new(MsgsTestSuite))
}

func (suite *MsgsTestSuite) SetupTest() {
	deployer := "cosmos1"
	withdraw := "cosmos2"
	suite.contract = sdk.AccAddress([]byte("nibi15u3dt79t6sxxa3x3kpkhzsy56edaa5a66wvt3kxmukqjz2sx0hes5sn38g"))
	suite.deployer = sdk.AccAddress([]byte(deployer))
	suite.deployerStr = suite.deployer.String()
	suite.withdrawerStr = sdk.AccAddress([]byte(withdraw)).String()
}

func (suite *MsgsTestSuite) TestMsgRegisterFeeShareGetters() {
	msg := NewMsgRegisterFeeShare(
		suite.contract,
		suite.deployer,
		suite.deployer,
	)
	suite.Require().Equal(RouterKey, msg.Route())
	suite.Require().Equal(TypeMsgRegisterFeeShare, msg.Type())
	suite.Require().NotNil(msg.GetSigners())
}

func (suite *MsgsTestSuite) TestMsgCancelFeeShareGetters() {
	msg := NewMsgCancelFeeShare(
		suite.contract,
		sdk.AccAddress(suite.deployer.Bytes()),
	)
	suite.Require().Equal(RouterKey, msg.Route())
	suite.Require().Equal(TypeMsgCancelFeeShare, msg.Type())
	suite.Require().NotNil(msg.GetSigners())
}

func (suite *MsgsTestSuite) TestMsgUpdateFeeShareGetters() {
	msg := NewMsgUpdateFeeShare(
		suite.contract,
		sdk.AccAddress(suite.deployer.Bytes()),
		sdk.AccAddress(suite.deployer.Bytes()),
	)
	suite.Require().Equal(RouterKey, msg.Route())
	suite.Require().Equal(TypeMsgUpdateFeeShare, msg.Type())
	suite.Require().NotNil(msg.GetSigners())
}

func (s *MsgsTestSuite) TestQuery_ValidateBasic() {
	validAddr := s.contract.String()
	invalidAddr := "invalid-addr"

	for _, tc := range []struct {
		name string
		test func()
	}{
		{
			name: "query fee share", test: func() {
				queryMsg := &QueryFeeShareRequest{
					ContractAddress: validAddr,
				}
				s.NoError(queryMsg.ValidateBasic())

				queryMsg.ContractAddress = invalidAddr
				s.Error(queryMsg.ValidateBasic())
			},
		},
		{
			name: "query fee shares", test: func() {
				queryMsg := &QueryFeeSharesRequest{
					Deployer: validAddr,
				}
				s.NoError(queryMsg.ValidateBasic())

				queryMsg.Deployer = invalidAddr
				s.Error(queryMsg.ValidateBasic())
			},
		},
		{
			name: "query fee shares by withdraw", test: func() {
				queryMsg := &QueryFeeSharesByWithdrawerRequest{
					WithdrawerAddress: validAddr,
				}
				s.NoError(queryMsg.ValidateBasic())

				queryMsg.WithdrawerAddress = invalidAddr
				s.Error(queryMsg.ValidateBasic())
			},
		},
	} {
		s.Run(tc.name, func() {
			tc.test()
		})
	}
}
