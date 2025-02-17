package ante_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/stretchr/testify/suite"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/app"
	nibiruante "github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
)

// AnteTestSuite is a test suite to be used with ante handler tests.
type AnteTestSuite struct {
	suite.Suite

	app         *app.NibiruApp
	anteHandler sdk.AnteHandler
	ctx         sdk.Context
	clientCtx   client.Context
	txBuilder   client.TxBuilder
}

// SetupTest setups a new test, with new app, context, and anteHandler.
func (suite *AnteTestSuite) SetupTest() {
	// Set up base app and ctx
	testapp.EnsureNibiruPrefix()
	encodingConfig := app.MakeEncodingConfig()
	suite.app = testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encodingConfig.Codec))
	chainId := "test-chain-id"
	ctx := suite.app.NewContext(true, tmproto.Header{
		Height:  1,
		ChainID: chainId,
		Time:    time.Now().UTC(),
	})
	suite.ctx = ctx

	// Set up TxConfig
	suite.clientCtx = client.Context{}.
		WithTxConfig(encodingConfig.TxConfig).
		WithChainID(chainId).
		WithLegacyAmino(encodingConfig.Amino)

	err := suite.app.AccountKeeper.SetParams(ctx, authtypes.DefaultParams())
	suite.Require().NoError(err)
	params := suite.app.AccountKeeper.GetParams(ctx)
	suite.Require().NoError(params.Validate())

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(suite.app.AccountKeeper),
		nibiruante.AnteDecoratorEnsureSinglePostPriceMessage{},
		nibiruante.AnteDecoratorStakingCommission{},
		ante.NewConsumeGasForTxSizeDecorator(suite.app.AccountKeeper),
		ante.NewDeductFeeDecorator(suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.FeeGrantKeeper, nil), // Replace fee ante from cosmos auth with a custom one.

		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(suite.app.AccountKeeper),
		ante.NewValidateSigCountDecorator(suite.app.AccountKeeper),
		ante.NewSigGasConsumeDecorator(suite.app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(suite.app.AccountKeeper, encodingConfig.TxConfig.SignModeHandler()),
		ante.NewIncrementSequenceDecorator(suite.app.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(suite.app.GetIBCKeeper()),
	}

	suite.anteHandler = sdk.ChainAnteDecorators(anteDecorators...)
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (suite *AnteTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (xauthsigning.Tx, error) {
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  suite.clientCtx.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
		acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress(priv.PubKey().Address()))
		err := acc.SetAccountNumber(uint64(i) + 100)
		suite.Require().NoError(err)
	}
	err := suite.txBuilder.SetSignatures(sigsV2...)
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
			suite.clientCtx.TxConfig.SignModeHandler().DefaultMode(), signerData,
			suite.txBuilder, priv, suite.clientCtx.TxConfig, accSeqs[i])
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}
