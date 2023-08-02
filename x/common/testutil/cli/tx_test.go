package cli_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/cli"

	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (s *IntegrationTestSuite) TestSendTx() {
	fromAddr := s.network.Validators[0].Address
	toAddr := testutil.AccAddress()
	sendCoin := sdk.NewCoin(denoms.NIBI, sdk.NewInt(42))
	txResp, err := s.network.BroadcastMsgs(fromAddr, &banktypes.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      sdk.NewCoins(sendCoin)},
	)
	s.NoError(err)
	s.EqualValues(0, txResp.Code)
}

func (s *IntegrationTestSuite) TestExecTx() {
	fromAddr := s.network.Validators[0].Address
	toAddr := testutil.AccAddress()
	sendCoin := sdk.NewCoin(denoms.NIBI, sdk.NewInt(69))
	args := []string{fromAddr.String(), toAddr.String(), sendCoin.String()}
	txResp, err := s.network.ExecTxCmd(bankcli.NewSendTxCmd(), fromAddr, args)
	s.NoError(err)
	s.EqualValues(0, txResp.Code)

	s.T().Run("test tx option changes", func(t *testing.T) {
		defaultOpts := cli.DEFAULT_TX_OPTIONS
		opts := cli.WithTxOptions(cli.TxOptionChanges{
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
		networkNoVals := new(cli.Network)
		*networkNoVals = *s.network
		networkNoVals.Validators = []*cli.Validator{}
		_, err := networkNoVals.ExecTxCmd(bankcli.NewTxCmd(), fromAddr, args)
		s.Error(err)
		s.Contains(err.Error(), "")
	})
}

func (s *IntegrationTestSuite) TestFillWalletFromValidator() {
	toAddr := testutil.AccAddress()
	val := s.network.Validators[0]
	funds := sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, 420),
	)
	feeDenom := denoms.NIBI
	s.NoError(cli.FillWalletFromValidator(
		toAddr, funds, val, feeDenom,
	))
}
