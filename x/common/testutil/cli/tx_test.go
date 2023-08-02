package cli_test

import (
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
	txResp, err := s.network.SendTx(fromAddr, &banktypes.MsgSend{
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

	txResp, err := cli.ExecTx(s.network, bankcli.NewSendTxCmd(), fromAddr, args)
	s.NoError(err)
	s.EqualValues(0, txResp.Code)

	// test tx option changes
	canFail := true
	opts := cli.WithTxOptions(cli.TxOptionChanges{CanFail: &canFail})
	txResp, err = cli.ExecTx(s.network, bankcli.NewSendTxCmd(), fromAddr, args, opts)
	s.NoError(err)
	s.EqualValues(0, txResp.Code)
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
