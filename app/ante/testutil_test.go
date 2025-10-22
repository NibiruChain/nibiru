package ante_test

import (
	"log"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	nibiruante "github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
)

func TestAnte(t *testing.T) {
	suite.Run(t, new(Suite))
}

// Suite is a test suite to be used with ante handler tests.
type Suite struct {
	testutil.LogRoutingSuite

	app         *app.NibiruApp
	anteHandler sdk.AnteHandler
	ctx         sdk.Context
	clientCtx   client.Context
	txBuilder   client.TxBuilder
}

// SetupTest setups a new test, with new app, context, and anteHandler.
func (s *Suite) SetupTest() {
	s.app, _ = testapp.NewNibiruTestApp(app.GenesisState{})
	chainId := "test-chain-id"
	baseapp.SetChainID(chainId)(s.app.BaseApp)
	ctx := s.app.NewContext(true, tmproto.Header{
		Height:  1,
		ChainID: chainId,
		Time:    time.Now().UTC(),
	})
	s.ctx = ctx

	// Set up TxConfig
	s.clientCtx = client.Context{}.
		WithTxConfig(s.app.GetTxConfig()).
		WithChainID(chainId).
		WithLegacyAmino(s.app.LegacyAmino())

	err := s.app.AccountKeeper.SetParams(ctx, authtypes.DefaultParams())
	s.Require().NoError(err)
	params := s.app.AccountKeeper.GetParams(ctx)
	s.Require().NoError(params.Validate())

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(s.app.AccountKeeper),
		nibiruante.AnteDecEnsureSinglePostPriceMessage{},
		nibiruante.AnteDecoratorStakingCommission{},
		ante.NewConsumeGasForTxSizeDecorator(s.app.AccountKeeper),
		ante.NewDeductFeeDecorator(s.app.AccountKeeper, s.app.BankKeeper, s.app.FeeGrantKeeper, nil), // Replace fee ante from cosmos auth with a custom one.

		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(s.app.AccountKeeper),
		ante.NewValidateSigCountDecorator(s.app.AccountKeeper),
		ante.NewSigGasConsumeDecorator(s.app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(s.app.AccountKeeper, s.app.GetTxConfig().SignModeHandler()),
		ante.NewIncrementSequenceDecorator(s.app.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(s.app.GetIBCKeeper()),
	}

	s.anteHandler = sdk.ChainAnteDecorators(anteDecorators...)
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (s *Suite) CreateTestTx(
	txBuilder client.TxBuilder,
	privs []cryptotypes.PrivKey,
	accNums []uint64,
	accSeqs []uint64,
	chainID string,
	txCfg client.TxConfig,
) (xauthsigning.Tx, error) {
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	log.Printf("s.clientCtx.ChainID: %v\n", s.clientCtx.ChainID)
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  txCfg.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
		// TODO: UD-DEBUG: REMOVED: Don't create new accounts here - use the existing ones
		// acc := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, sdk.AccAddress(priv.PubKey().Address()))
		// err := acc.SetAccountNumber(uint64(i) + 100)
		// s.Require().NoError(err)
	}
	err := txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			txCfg.SignModeHandler().DefaultMode(),
			signerData,
			txBuilder,
			priv,
			txCfg,
			accSeqs[i],
		)
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}
