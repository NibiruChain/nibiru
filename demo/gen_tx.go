package main

import (
	"encoding/base64"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

// from https://docs.cosmos.network/master/run-node/txs.html#programmatically-with-go
func main() {
	// Choose your codec: Amino or Protobuf. Here, we use Protobuf, given by the
	// following function.
	encCfg := simapp.MakeTestEncodingConfig()

	// Create a new TxBuilder.
	txBuilder := encCfg.TxConfig.NewTxBuilder()

	// priv1, _, addr1 := testdata.KeyTestPubAddr()
	// priv2, _, addr2 := testdata.KeyTestPubAddr()
	// _, _, addr3 := testdata.KeyTestPubAddr()

	privKeys, addrs := sample.PrivKeyAddressPairs(3)

	// Define two x/bank MsgSend messages:
	// - from addr1 to addr3,
	// - from addr2 to addr3.
	// This means that the transactions needs two signers: addr1 and addr2.
	msg1 := banktypes.NewMsgSend(addrs[0], addrs[2], types.NewCoins(types.NewInt64Coin("atom", 12)))
	msg2 := banktypes.NewMsgSend(addrs[1], addrs[2], types.NewCoins(types.NewInt64Coin("atom", 34)))

	err := txBuilder.SetMsgs(msg1, msg2)
	if err != nil {
		panic(err)
	}

	txBuilder.SetGasLimit(testdata.NewTestGasLimit())

	privs := []cryptotypes.PrivKey{privKeys[0], privKeys[1]}
	accountNumbers := []uint64{0, 0} // The accounts' account numbers
	sequences := []uint64{0, 0}      // The accounts' sequence numbers

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var signatures []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: sequences[i],
		}

		signatures = append(signatures, sigV2)
	}
	err = txBuilder.SetSignatures(signatures...)
	if err != nil {
		panic(err)
	}

	// Second round: all signer infos are set, so each signer can sign.
	signatures = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := authsigning.SignerData{
			ChainID:       "nibiru-localnet-0",
			AccountNumber: accountNumbers[i],
			Sequence:      sequences[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			encCfg.TxConfig.SignModeHandler().DefaultMode(), signerData,
			txBuilder, priv, encCfg.TxConfig, sequences[i])

		if err != nil {
			panic(err)
		}

		signatures = append(signatures, sigV2)
	}
	err = txBuilder.SetSignatures(signatures...)
	if err != nil {
		panic(err)
	}

	// Generated Protobuf-encoded bytes.
	txBytes, err := encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		panic(err)
	}

	txBase64 := base64.StdEncoding.EncodeToString(txBytes)
	fmt.Println(txBase64)

	// Generate a JSON string.
	txJSONBytes, err := encCfg.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	if err != nil {
		panic(err)
	}
	fmt.Println(string(txJSONBytes))
}
