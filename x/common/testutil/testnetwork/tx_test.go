package testnetwork_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testnetwork"
)

func (s *TestSuite) TestSendTx() {
	fromAddr := s.network.Validators[0].Address
	toAddr := testutil.AccAddress()
	sendCoin := sdk.NewCoin(denoms.NIBI, math.NewInt(42))
	txResp, err := s.network.BroadcastMsgs(fromAddr, &banktypes.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      sdk.NewCoins(sendCoin),
	},
	)
	s.NoError(err)
	s.EqualValues(0, txResp.Code)
}

func (s *TestSuite) TestExecTx() {
	fromAddr := s.network.Validators[0].Address
	toAddr := testutil.AccAddress()
	sendCoin := sdk.NewCoin(denoms.NIBI, math.NewInt(69))
	args := []string{fromAddr.String(), toAddr.String(), sendCoin.String()}
	txResp, err := s.network.ExecTxCmd(bankcli.NewSendTxCmd(), fromAddr, args)
	s.NoError(err)
	s.EqualValues(0, txResp.Code)

	s.T().Run("test tx option changes", func(t *testing.T) {
		defaultOpts := testnetwork.DEFAULT_TX_OPTIONS
		opts := testnetwork.WithTxOptions(testnetwork.TxOptionChanges{
			BroadcastMode:    &defaultOpts.BroadcastMode,
			CanFail:          &defaultOpts.CanFail,
			Fees:             &defaultOpts.Fees,
			Gas:              &defaultOpts.Gas,
			KeyringBackend:   &defaultOpts.KeyringBackend,
			SkipConfirmation: &defaultOpts.SkipConfirmation,
		})
		txResp, err = s.network.ExecTxCmd(bankcli.NewSendTxCmd(), fromAddr, args, opts)
		s.NoError(err)
		s.EqualValues(0, txResp.Code)
	})

	s.T().Run("fail when validators are missing", func(t *testing.T) {
		networkNoVals := new(testnetwork.Network)
		*networkNoVals = *s.network
		networkNoVals.Validators = []*testnetwork.Validator{}
		_, err := networkNoVals.ExecTxCmd(bankcli.NewTxCmd(), fromAddr, args)
		s.Error(err)
		s.Contains(err.Error(), "")
	})
}

func (s *TestSuite) TestFillWalletFromValidator() {
	toAddr := testutil.AccAddress()
	val := s.network.Validators[0]
	funds := sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, 420),
	)
	feeDenom := denoms.NIBI
	s.NoError(testnetwork.FillWalletFromValidator(
		toAddr, funds, val, feeDenom,
	))
}
