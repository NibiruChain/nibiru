package eip712_test

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec/types"

	chainparams "cosmossdk.io/simapp/params"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth/eip712"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/eth/encoding"

	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
)

// Unit tests for single-signer EIP-712 signature verification. Multi-signature key verification tests are out-of-scope
// here and included with the ante_tests.

const (
	msgsFieldName    = "msgs"
	TESTNET_CHAIN_ID = "nibiru_9000"
)

type EIP712TestSuite struct {
	suite.Suite

	encCfg                   chainparams.EncodingConfig
	clientCtx                client.Context
	useLegacyEIP712TypedData bool
	denom                    string
}

type EIP712TestParams struct {
	fee           sdktx.Fee
	address       sdk.AccAddress
	accountNumber uint64
	sequence      uint64
	memo          string
}

func TestEIP712TestSuite(t *testing.T) {
	suite.Run(t, &EIP712TestSuite{})
	// Note that we don't test the Legacy EIP-712 Extension, since that case
	// is sufficiently covered by the AnteHandler tests.
	suite.Run(t, &EIP712TestSuite{
		useLegacyEIP712TypedData: true,
	})
}

func (suite *EIP712TestSuite) SetupTest() {
	suite.encCfg = encoding.MakeConfig()
	suite.clientCtx = client.Context{}.WithTxConfig(suite.encCfg.TxConfig)
	suite.denom = evm.EVMBankDenom

	eip712.SetEncodingConfig(suite.encCfg)
}

// createTestAddress creates random test addresses for messages
func (suite *EIP712TestSuite) createTestAddress() sdk.AccAddress {
	privkey, _ := ethsecp256k1.GenerateKey()
	key, err := privkey.ToECDSA()
	suite.Require().NoError(err)

	addr := crypto.PubkeyToAddress(key.PublicKey)

	return addr.Bytes()
}

// createTestKeyPair creates a random keypair for signing and verification
func (suite *EIP712TestSuite) createTestKeyPair() (*ethsecp256k1.PrivKey, *ethsecp256k1.PubKey) {
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	pubKey := &ethsecp256k1.PubKey{
		Key: privKey.PubKey().Bytes(),
	}
	suite.Require().Implements((*cryptotypes.PubKey)(nil), pubKey)

	return privKey, pubKey
}

// makeCoins helps create an instance of sdk.Coins[] with single coin
func (suite *EIP712TestSuite) makeCoins(denom string, amount sdkmath.Int) sdk.Coins {
	return sdk.NewCoins(
		sdk.NewCoin(
			denom,
			amount,
		),
	)
}

func (suite *EIP712TestSuite) TestEIP712() {
	suite.SetupTest()

	signModes := []signing.SignMode{
		signing.SignMode_SIGN_MODE_DIRECT,
		signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
	}

	params := EIP712TestParams{
		fee: sdktx.Fee{
			Amount:   suite.makeCoins(suite.denom, sdkmath.NewInt(2000)),
			GasLimit: 20000,
		},
		address:       suite.createTestAddress(),
		accountNumber: 25,
		sequence:      78,
		memo:          "",
	}

	testCases := []struct {
		title         string
		chainID       string
		msgs          []sdk.Msg
		timeoutHeight uint64
		expectSuccess bool
	}{
		{
			title: "Succeeds - Standard MsgSend",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(
					suite.createTestAddress(),
					suite.createTestAddress(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(1)),
				),
			},
			expectSuccess: true,
		},
		{
			title: "Succeeds - Standard MsgVote",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
			},
			expectSuccess: true,
		},
		{
			title: "Succeeds - Standard MsgDelegate",
			msgs: []sdk.Msg{
				stakingtypes.NewMsgDelegate(
					suite.createTestAddress(),
					sdk.ValAddress(suite.createTestAddress()),
					suite.makeCoins(suite.denom, sdkmath.NewInt(1))[0],
				),
			},
			expectSuccess: true,
		},
		{
			title: "Succeeds - Standard MsgWithdrawDelegationReward",
			msgs: []sdk.Msg{
				distributiontypes.NewMsgWithdrawDelegatorReward(
					suite.createTestAddress(),
					sdk.ValAddress(suite.createTestAddress()),
				),
			},
			expectSuccess: true,
		},
		{
			title: "Succeeds - Two Single-Signer MsgDelegate",
			msgs: []sdk.Msg{
				stakingtypes.NewMsgDelegate(
					params.address,
					sdk.ValAddress(suite.createTestAddress()),
					suite.makeCoins(suite.denom, sdkmath.NewInt(1))[0],
				),
				stakingtypes.NewMsgDelegate(
					params.address,
					sdk.ValAddress(suite.createTestAddress()),
					suite.makeCoins(suite.denom, sdkmath.NewInt(5))[0],
				),
			},
			expectSuccess: true,
		},
		{
			title: "Succeeds - Single-Signer MsgVote V1 with Omitted Value",
			msgs: []sdk.Msg{
				govtypesv1.NewMsgVote(
					params.address,
					5,
					govtypesv1.VoteOption_VOTE_OPTION_NO,
					"",
				),
			},
			expectSuccess: true,
		},
		{
			title: "Succeeds - Single-Signer MsgSend + MsgVote",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					params.address,
					5,
					govtypes.OptionNo,
				),
				banktypes.NewMsgSend(
					params.address,
					suite.createTestAddress(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(50)),
				),
			},
			expectSuccess: !suite.useLegacyEIP712TypedData,
		},
		{
			title: "Succeeds - Single-Signer 2x MsgVoteV1 with Different Schemas",
			msgs: []sdk.Msg{
				govtypesv1.NewMsgVote(
					params.address,
					5,
					govtypesv1.VoteOption_VOTE_OPTION_NO,
					"",
				),
				govtypesv1.NewMsgVote(
					params.address,
					10,
					govtypesv1.VoteOption_VOTE_OPTION_YES,
					"Has Metadata",
				),
			},
			expectSuccess: !suite.useLegacyEIP712TypedData,
		},
		{
			title: "Fails - Two MsgVotes with Different Signers",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					25,
					govtypes.OptionAbstain,
				),
			},
			expectSuccess: false,
		},
		{
			title:         "Fails - Empty Transaction",
			msgs:          []sdk.Msg{},
			expectSuccess: false,
		},
		{
			title:   "Success - Invalid ChainID uses default",
			chainID: "invalidchainid",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
			},
			expectSuccess: true,
		},
		{
			title: "Fails - Includes TimeoutHeight",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
			},
			timeoutHeight: 1000,
			expectSuccess: false,
		},
		{
			title: "Fails - Single Message / Multi-Signer",
			msgs: []sdk.Msg{
				banktypes.NewMsgMultiSend(
					[]banktypes.Input{
						banktypes.NewInput(
							suite.createTestAddress(),
							suite.makeCoins(suite.denom, sdkmath.NewInt(50)),
						),
						banktypes.NewInput(
							suite.createTestAddress(),
							suite.makeCoins(suite.denom, sdkmath.NewInt(50)),
						),
					},
					[]banktypes.Output{
						banktypes.NewOutput(
							suite.createTestAddress(),
							suite.makeCoins(suite.denom, sdkmath.NewInt(50)),
						),
						banktypes.NewOutput(
							suite.createTestAddress(),
							suite.makeCoins(suite.denom, sdkmath.NewInt(50)),
						),
					},
				),
			},
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		for _, signMode := range signModes {
			suite.Run(tc.title, func() {
				privKey, pubKey := suite.createTestKeyPair()

				txBuilder := suite.clientCtx.TxConfig.NewTxBuilder()

				txBuilder.SetGasLimit(params.fee.GasLimit)
				txBuilder.SetFeeAmount(params.fee.Amount)

				err := txBuilder.SetMsgs(tc.msgs...)
				suite.Require().NoError(err)

				txBuilder.SetMemo(params.memo)

				// Prepare signature field with empty signatures
				txSigData := signing.SingleSignatureData{
					SignMode:  signMode,
					Signature: nil,
				}
				txSig := signing.SignatureV2{
					PubKey:   pubKey,
					Data:     &txSigData,
					Sequence: params.sequence,
				}

				err = txBuilder.SetSignatures([]signing.SignatureV2{txSig}...)
				suite.Require().NoError(err)

				chainID := TESTNET_CHAIN_ID + "-1"
				if tc.chainID != "" {
					chainID = tc.chainID
				}

				if tc.timeoutHeight != 0 {
					txBuilder.SetTimeoutHeight(tc.timeoutHeight)
				}

				signerData := authsigning.SignerData{
					ChainID:       chainID,
					AccountNumber: params.accountNumber,
					Sequence:      params.sequence,
					PubKey:        pubKey,
					Address:       sdk.MustBech32ifyAddressBytes(appconst.AccountAddressPrefix, pubKey.Bytes()),
				}

				bz, err := suite.clientCtx.TxConfig.SignModeHandler().GetSignBytes(
					signMode,
					signerData,
					txBuilder.GetTx(),
				)
				suite.Require().NoError(err)

				suite.verifyEIP712SignatureVerification(tc.expectSuccess, *privKey, *pubKey, bz)

				// Verify payload flattening only if the payload is in valid JSON format
				if signMode == signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON {
					suite.verifySignDocFlattening(bz)

					if tc.expectSuccess {
						suite.verifyBasicTypedData(bz)
					}
				}
			})
		}
	}
}

// verifyEIP712SignatureVerification verifies that the payload passes signature verification if signed as its EIP-712 representation.
func (suite *EIP712TestSuite) verifyEIP712SignatureVerification(expectedSuccess bool, privKey ethsecp256k1.PrivKey, pubKey ethsecp256k1.PubKey, signBytes []byte) {
	eip712Bytes, err := eip712.GetEIP712BytesForMsg(signBytes)

	if suite.useLegacyEIP712TypedData {
		eip712Bytes, err = eip712.LegacyGetEIP712BytesForMsg(signBytes)
	}

	if !expectedSuccess {
		suite.Require().Error(err)
		return
	}

	suite.Require().NoError(err)

	sig, err := privKey.Sign(eip712Bytes)
	suite.Require().NoError(err)

	// Verify against original payload bytes. This should pass, even though it is not
	// the original message that was signed.
	res := pubKey.VerifySignature(signBytes, sig)
	suite.Require().True(res)

	// Verify against the signed EIP-712 bytes. This should pass, since it is the message signed.
	res = pubKey.VerifySignature(eip712Bytes, sig)
	suite.Require().True(res)

	// Verify against random bytes to ensure it does not pass unexpectedly (sanity check).
	randBytes := make([]byte, len(signBytes))
	copy(randBytes, signBytes)
	// Change the first element of signBytes to a different value
	randBytes[0] = (signBytes[0] + 10) % 255
	res = pubKey.VerifySignature(randBytes, sig)
	suite.Require().False(res)
}

// verifySignDocFlattening tests the flattening algorithm against the sign doc's JSON payload,
// using verifyPayloadAgainstFlattened.
func (suite *EIP712TestSuite) verifySignDocFlattening(signDoc []byte) {
	payload := gjson.ParseBytes(signDoc)
	suite.Require().True(payload.IsObject())

	flattened, _, err := eip712.FlattenPayloadMessages(payload)
	suite.Require().NoError(err)

	suite.verifyPayloadAgainstFlattened(payload, flattened)
}

// verifyPayloadAgainstFlattened compares a payload against its flattened counterpart to ensure that
// the flattening algorithm behaved as expected.
func (suite *EIP712TestSuite) verifyPayloadAgainstFlattened(payload gjson.Result, flattened gjson.Result) {
	payloadMap, ok := payload.Value().(map[string]interface{})
	suite.Require().True(ok)
	flattenedMap, ok := flattened.Value().(map[string]interface{})
	suite.Require().True(ok)

	suite.verifyPayloadMapAgainstFlattenedMap(payloadMap, flattenedMap)
}

// verifyPayloadMapAgainstFlattenedMap directly compares two JSON maps in Go representations to
// test flattening.
func (suite *EIP712TestSuite) verifyPayloadMapAgainstFlattenedMap(original map[string]interface{}, flattened map[string]interface{}) {
	interfaceMessages, ok := original[msgsFieldName]
	suite.Require().True(ok)

	messages, ok := interfaceMessages.([]interface{})
	suite.Require().True(ok)

	// Verify message contents
	for i, msg := range messages {
		flattenedMsg, ok := flattened[fmt.Sprintf("msg%d", i)]
		suite.Require().True(ok)

		flattenedMsgJSON, ok := flattenedMsg.(map[string]interface{})
		suite.Require().True(ok)

		suite.Require().Equal(flattenedMsgJSON, msg)
	}

	// Verify new payload does not have msgs field
	_, ok = flattened[msgsFieldName]
	suite.Require().False(ok)

	// Verify number of total keys
	numKeysOriginal := len(original)
	numKeysFlattened := len(flattened)
	numMessages := len(messages)

	// + N keys, then -1 for msgs
	suite.Require().Equal(numKeysFlattened, numKeysOriginal+numMessages-1)

	// Verify contents of remaining keys
	for k, obj := range original {
		if k == msgsFieldName {
			continue
		}

		flattenedObj, ok := flattened[k]
		suite.Require().True(ok)

		suite.Require().Equal(obj, flattenedObj)
	}
}

// verifyBasicTypedData performs basic verification on the TypedData generation.
func (suite *EIP712TestSuite) verifyBasicTypedData(signDoc []byte) {
	typedData, err := eip712.GetEIP712TypedDataForMsg(signDoc)

	suite.Require().NoError(err)

	jsonPayload := gjson.ParseBytes(signDoc)
	suite.Require().True(jsonPayload.IsObject())

	flattened, _, err := eip712.FlattenPayloadMessages(jsonPayload)
	suite.Require().NoError(err)
	suite.Require().True(flattened.IsObject())

	flattenedMsgMap, ok := flattened.Value().(map[string]interface{})
	suite.Require().True(ok)

	suite.Require().Equal(typedData.Message, flattenedMsgMap)
}

// TestFlattenPayloadErrorHandling tests error handling in TypedData generation,
// specifically regarding the payload.
func (suite *EIP712TestSuite) TestFlattenPayloadErrorHandling() {
	// No msgs
	_, _, err := eip712.FlattenPayloadMessages(gjson.Parse(""))
	suite.Require().ErrorContains(err, "no messages found")

	// Non-array Msgs
	_, _, err = eip712.FlattenPayloadMessages(gjson.Parse(`{"msgs": 10}`))
	suite.Require().ErrorContains(err, "array of messages")

	// Array with non-object items
	_, _, err = eip712.FlattenPayloadMessages(gjson.Parse(`{"msgs": [10, 20]}`))
	suite.Require().ErrorContains(err, "not valid JSON")

	// Malformed payload
	malformed, err := sjson.Set(suite.generateRandomPayload(2).Raw, "msg0", 20)
	suite.Require().NoError(err)
	_, _, err = eip712.FlattenPayloadMessages(gjson.Parse(malformed))
	suite.Require().ErrorContains(err, "malformed payload")
}

// TestTypedDataErrorHandling tests error handling for TypedData generation
// in the main algorithm.
func (suite *EIP712TestSuite) TestTypedDataErrorHandling() {
	// Empty JSON
	_, err := eip712.WrapTxToTypedData(0, make([]byte, 0))
	suite.Require().ErrorContains(err, "invalid JSON")

	_, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": 10}`).Raw))
	suite.Require().ErrorContains(err, "array of messages")

	// Invalid message 'type'
	_, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": 10 }] }`).Raw))
	suite.Require().ErrorContains(err, "message type value")

	// Max duplicate type recursion depth
	messagesArr := new(bytes.Buffer)
	maxRecursionDepth := 1001

	messagesArr.WriteString("[")
	for i := 0; i < maxRecursionDepth; i++ {
		fmt.Fprintf(messagesArr, `{ "type": "msgType", "value": { "field%v": 10 } }`, i)
		if i != maxRecursionDepth-1 {
			messagesArr.WriteString(",")
		}
	}
	messagesArr.WriteString("]")

	_, err = eip712.WrapTxToTypedData(0, []byte(fmt.Sprintf(`{ "msgs": %v }`, messagesArr)))
	suite.Require().ErrorContains(err, "maximum number of duplicates")
}

// TestTypedDataEdgeCases tests certain interesting edge cases to ensure that they work
// (or don't work) as expected.
func (suite *EIP712TestSuite) TestTypedDataEdgeCases() {
	// Type without '/' separator
	typedData, err := eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": 10 } }] }`).Raw))
	suite.Require().NoError(err)
	types := typedData.Types["TypeMsgSend0"]
	suite.Require().Greater(len(types), 0)

	// Null value
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": null } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	// Skip null type, since we don't expect any in the payload
	suite.Require().Equal(len(types), 0)

	// Boolean value
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": true } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(len(types), 1)
	suite.Require().Equal(types[0], apitypes.Type{
		Name: "field",
		Type: "bool",
	})

	// Empty array
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": [] } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(types[0], apitypes.Type{
		Name: "field",
		Type: "string[]",
	})

	// Simple arrays
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "array": [1, 2, 3] } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(len(types), 1)
	suite.Require().Equal(types[0], apitypes.Type{
		Name: "array",
		Type: "int64[]",
	})

	// Nested arrays (EIP-712 does not support nested arrays)
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "array": [[1, 2, 3], [1, 2]] } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(len(types), 0)
}

// TestTypedDataGeneration tests certain qualities about the output Types representation.
func (s *EIP712TestSuite) TestTypedDataGeneration() {
	// Multiple messages with the same schema should share one type
	payloadRaw := `{ "msgs": [{ "type": "msgType", "value": { "field1": 10 }}, { "type": "msgType", "value": { "field1": 20 }}] }`

	typedData, err := eip712.WrapTxToTypedData(0, []byte(payloadRaw))
	s.Require().NoError(err)
	s.Require().True(typedData.Types["TypemsgType1"] == nil)

	// Multiple messages with different schemas should have different types
	payloadRaw = `{ "msgs": [{ "type": "msgType", "value": { "field1": 10 }}, { "type": "msgType", "value": { "field2": 20 }}] }`

	typedData, err = eip712.WrapTxToTypedData(0, []byte(payloadRaw))
	s.Require().NoError(err)
	s.Require().False(typedData.Types["TypemsgType1"] == nil)
}

func (s *EIP712TestSuite) TestTypToEth() {
	cases := []struct {
		want  string
		given any
	}{
		{want: "string", given: "string"},
		{want: "int8", given: int8(0)},
		{want: "int16", given: int16(0)},
		{want: "int32", given: int32(0)},
		{want: "int64", given: int64(0)},

		{want: "uint64", given: uint(0)},
		{want: "uint8", given: uint8(0)},
		{want: "uint16", given: uint16(0)},
		{want: "uint32", given: uint32(0)},
		{want: "uint64", given: uint64(0)},
		{want: "bool", given: false},

		// slice and array cases
		{want: "uint64[]", given: []uint64{1, 2, 3}},
		{want: "string[]", given: []string{"1", "2"}},
		{want: "int8[]", given: [3]int8{3, 2, 1}},

		// pointer cases
		{want: "string", given: sdkmath.NewInt(1)},
		{want: "string", given: big.NewInt(1)},
		{want: "string", given: sdkmath.LegacyNewDec(1)},
	}

	for _, tc := range cases {
		fnInp := reflect.TypeOf(tc.given)
		result := eip712.TypToEth(fnInp)
		s.Equal(tc.want, result,
			"Type conversion did not match for %v with input %s", tc.given, fnInp)
	}
}

func (s *EIP712TestSuite) TestUnpackAny() {
	_, addr := testutil.PrivKey()
	cases := []struct {
		wantWrappedType string
		wantType        string
		given           sdk.Msg
		wantErr         bool
	}{
		{
			wantWrappedType: "*eip712.CosmosAnyWrapper",
			wantType:        "/cosmos.bank.v1beta1.MsgSend",
			given:           banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewInt64Coin("unibi", 25))),
		},
		{
			wantWrappedType: "*eip712.CosmosAnyWrapper",
			wantType:        "/eth.evm.v1.MsgEthereumTx",
			given:           new(evm.MsgEthereumTx),
		},
		{
			given:   nil,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		anyGiven, err := sdkcodec.NewAnyWithValue(tc.given)
		if tc.wantErr {
			s.Require().Error(err)
			continue
		}
		s.NoError(err)

		reflectVal := reflect.ValueOf(anyGiven)
		gotReflectType, gotReflectVal, err := eip712.UnpackAny(s.encCfg.Codec, reflectVal)
		s.Require().NoError(err,
			"got reflect.Type %s, got reflect.Value %s",
			gotReflectType, gotReflectVal)

		s.Equal(tc.wantWrappedType, gotReflectType.String())
		if gotWrappedAny := gotReflectVal.Interface().(*eip712.CosmosAnyWrapper); gotWrappedAny != nil {
			s.EqualValues(gotWrappedAny.Type, tc.wantType)
		}
	}
}
