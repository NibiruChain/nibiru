// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdkioerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/NibiruChain/nibiru/v2/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	_ sdk.Msg    = &MsgEthereumTx{}
	_ sdk.Tx     = &MsgEthereumTx{}
	_ ante.GasTx = &MsgEthereumTx{}
	_ sdk.Msg    = &MsgUpdateParams{}
	_ sdk.Msg    = &MsgCreateFunToken{}
	_ sdk.Msg    = &MsgConvertCoinToEvm{}

	_ codectypes.UnpackInterfacesMessage = MsgEthereumTx{}
)

// NewTx returns a reference to a new Ethereum transaction message.
func NewTx(
	tx *EvmTxArgs,
) *MsgEthereumTx {
	var (
		cid, amt, gp *sdkmath.Int
		toAddr       string
		txData       TxData
	)

	if tx.To != nil {
		toAddr = tx.To.Hex()
	}

	if tx.Amount != nil {
		amountInt := sdkmath.NewIntFromBigInt(tx.Amount)
		amt = &amountInt
	}

	if tx.ChainID != nil {
		chainIDInt := sdkmath.NewIntFromBigInt(tx.ChainID)
		cid = &chainIDInt
	}

	if tx.GasPrice != nil {
		gasPriceInt := sdkmath.NewIntFromBigInt(tx.GasPrice)
		gp = &gasPriceInt
	}

	switch {
	case tx.GasFeeCap != nil:
		gtc := sdkmath.NewIntFromBigInt(tx.GasTipCap)
		gfc := sdkmath.NewIntFromBigInt(tx.GasFeeCap)

		txData = &DynamicFeeTx{
			ChainID:   cid,
			Amount:    amt,
			To:        toAddr,
			GasTipCap: &gtc,
			GasFeeCap: &gfc,
			Nonce:     tx.Nonce,
			GasLimit:  tx.GasLimit,
			Data:      tx.Input,
			Accesses:  NewAccessList(tx.Accesses),
		}
	case tx.Accesses != nil:
		txData = &AccessListTx{
			ChainID:  cid,
			Nonce:    tx.Nonce,
			To:       toAddr,
			Amount:   amt,
			GasLimit: tx.GasLimit,
			GasPrice: gp,
			Data:     tx.Input,
			Accesses: NewAccessList(tx.Accesses),
		}
	default:
		txData = &LegacyTx{
			To:       toAddr,
			Amount:   amt,
			GasPrice: gp,
			Nonce:    tx.Nonce,
			GasLimit: tx.GasLimit,
			Data:     tx.Input,
		}
	}

	dataAny, err := PackTxData(txData)
	if err != nil {
		panic(err)
	}

	msg := MsgEthereumTx{Data: dataAny}
	msg.Hash = msg.AsTransaction().Hash().Hex()
	return &msg
}

// FromEthereumTx populates the message fields from the given ethereum transaction
func (msg *MsgEthereumTx) FromEthereumTx(tx *gethcore.Transaction) error {
	txData, err := NewTxDataFromTx(tx)
	if err != nil {
		return err
	}

	anyTxData, err := PackTxData(txData)
	if err != nil {
		return err
	}

	msg.Data = anyTxData
	msg.Hash = tx.Hash().Hex()
	return nil
}

// Route returns the route value of an MsgEthereumTx.
func (msg MsgEthereumTx) Route() string { return RouterKey }

func (msg MsgEthereumTx) Type() string { return proto.MessageName(new(MsgEthereumTx)) }

// ValidateBasic implements the sdk.Msg interface. It performs basic validation
// checks of a Transaction. If returns an error if validation fails.
func (msg MsgEthereumTx) ValidateBasic() error {
	if msg.From != "" {
		if err := eth.ValidateAddress(msg.From); err != nil {
			return sdkioerrors.Wrap(err, "invalid from address")
		}
	}

	// Validate Size_ field, should be kept empty
	if msg.Size_ != 0 {
		return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "tx size is deprecated")
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to unpack tx data")
	}

	gas := txData.GetGas()

	// prevent txs with 0 gas to fill up the mempool
	if gas == 0 {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidGasLimit, "gas limit must not be zero")
	}

	// prevent gas limit from overflow
	if g := new(big.Int).SetUint64(gas); !g.IsInt64() {
		return sdkioerrors.Wrap(core.ErrGasUintOverflow, "gas limit must be less than math.MaxInt64")
	}

	if err := txData.Validate(); err != nil {
		return sdkioerrors.Wrap(err, "failed \"TxData.Validate\"")
	}

	// Validate EthHash field after validated txData to avoid panic
	txHash := msg.AsTransaction().Hash().Hex()
	if msg.Hash != txHash {
		return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid tx hash %s, expected: %s", msg.Hash, txHash)
	}

	return nil
}

// GetMsgs returns a single MsgEthereumTx as sdk.Msg.
func (msg *MsgEthereumTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{msg}
}

// GetSigners returns the expected signers for an Ethereum transaction message.
// For such a message, there should exist only a single 'signer'.
//
// NOTE: This method panics if 'Sign' hasn't been called first.
func (msg *MsgEthereumTx) GetSigners() []sdk.AccAddress {
	data, err := UnpackTxData(msg.Data)
	if err != nil {
		panic(err)
	}

	sender, err := msg.GetSender(data.GetChainID())
	if err != nil {
		panic(err)
	}

	signer := sdk.AccAddress(sender.Bytes())
	return []sdk.AccAddress{signer}
}

// GetSignBytes returns the Amino bytes of an Ethereum transaction message used
// for signing.
//
// NOTE: This method cannot be used as a chain ID is needed to create valid bytes
// to sign over. Use 'RLPSignBytes' instead.
func (msg MsgEthereumTx) GetSignBytes() []byte {
	panic("must use 'RLPSignBytes' with a chain ID to get the valid bytes to sign")
}

// Sign calculates a secp256k1 ECDSA signature and signs the transaction. It
// takes a keyring signer and the chainID to sign an Ethereum transaction according to
// EIP155 standard.
// This method mutates the transaction as it populates the V, R, S
// fields of the Transaction's Signature.
// The function will fail if the sender address is not defined for the msg or if
// the sender is not registered on the keyring
func (msg *MsgEthereumTx) Sign(ethSigner gethcore.Signer, keyringSigner keyring.Signer) error {
	from := msg.GetFrom()
	if from.Empty() {
		return fmt.Errorf("sender address not defined for message")
	}

	tx := msg.AsTransaction()
	txHash := ethSigner.Hash(tx)

	sig, _, err := keyringSigner.SignByAddress(from, txHash.Bytes())
	if err != nil {
		return err
	}

	tx, err = tx.WithSignature(ethSigner, sig)
	if err != nil {
		return err
	}

	return msg.FromEthereumTx(tx)
}

// GetGas implements the GasTx interface. It returns the GasLimit of the transaction.
func (msg MsgEthereumTx) GetGas() uint64 {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return 0
	}
	return txData.GetGas()
}

// GetFee returns the fee for non dynamic fee tx
func (msg MsgEthereumTx) GetFee() *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.Fee()
}

// EffectiveFeeWei returns the fee for dynamic fee tx
func (msg MsgEthereumTx) EffectiveFeeWei(baseFee *big.Int) *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.EffectiveFeeWei(baseFee)
}

// EffectiveGasPriceWeiPerGas returns the effective gas price according to the base
// fee. This value is in units of "wei per unit gas".
func (msg MsgEthereumTx) EffectiveGasPriceWeiPerGas(baseFeeWei *big.Int) *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.EffectiveGasPriceWeiPerGas(baseFeeWei)
}

func (msg MsgEthereumTx) EffectiveGasCapWei(baseFeeWei *big.Int) *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.EffectiveGasFeeCapWei(baseFeeWei)
}

// GetFrom loads the ethereum sender address from the sigcache and returns an
// sdk.AccAddress from its bytes
func (msg *MsgEthereumTx) GetFrom() sdk.AccAddress {
	if msg.From == "" {
		return nil
	}

	return common.HexToAddress(msg.From).Bytes()
}

// AsTransaction creates an Ethereum Transaction type from the msg fields
func (msg MsgEthereumTx) AsTransaction() *gethcore.Transaction {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}

	return gethcore.NewTx(txData.AsEthereumData())
}

// AsMessage creates an Ethereum core.Message from the msg fields
func (msg MsgEthereumTx) AsMessage(
	signer gethcore.Signer,
	baseFeeWei *big.Int,
) (*core.Message, error) {
	return core.TransactionToMessage(msg.AsTransaction(), signer, baseFeeWei)
}

// GetSender extracts the sender address from the signature values using the latest signer for the given chainID.
func (msg *MsgEthereumTx) GetSender(chainID *big.Int) (common.Address, error) {
	signer := gethcore.LatestSignerForChainID(chainID)
	from, err := signer.Sender(msg.AsTransaction())
	if err != nil {
		return common.Address{}, err
	}

	msg.From = from.Hex()
	return from, nil
}

// UnpackInterfaces implements UnpackInterfacesMesssage.UnpackInterfaces
func (msg MsgEthereumTx) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return unpacker.UnpackAny(msg.Data, new(TxData))
}

// UnmarshalBinary decodes the canonical encoding of transactions.
func (msg *MsgEthereumTx) UnmarshalBinary(b []byte) error {
	tx := &gethcore.Transaction{}
	if err := tx.UnmarshalBinary(b); err != nil {
		return err
	}
	return msg.FromEthereumTx(tx)
}

// BuildTx builds the Cosmos-SDK [signing.Tx] from ethereum tx ([MsgEthereumTx])
func (msg *MsgEthereumTx) BuildTx(b client.TxBuilder, evmDenom string) (signing.Tx, error) {
	builder, ok := b.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errors.New("unsupported builder")
	}

	option, err := codectypes.NewAnyWithValue(&ExtensionOptionsEthereumTx{})
	if err != nil {
		return nil, err
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil, err
	}

	// Compute fees using effective fee to enforce 1unibi minimum gas price
	fees := make(sdk.Coins, 0)
	effectiveFeeMicronibi := WeiToNative(txData.EffectiveFeeWei(BASE_FEE_WEI))
	feeAmtMicronibi := sdkmath.NewIntFromBigInt(effectiveFeeMicronibi)
	if feeAmtMicronibi.Sign() > 0 {
		fees = append(fees, sdk.NewCoin(evmDenom, feeAmtMicronibi))
	}

	builder.SetExtensionOptions(option)

	// A valid msg should have empty `From`
	msg.From = ""

	err = builder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}
	builder.SetFeeAmount(fees)
	builder.SetGasLimit(msg.GetGas())
	tx := builder.GetTx()
	return tx, nil
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	//#nosec G703 -- gosec raises a warning about a non-handled error which we deliberately ignore here
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkioerrors.Wrap(err, "invalid authority address")
	}

	return m.Params.Validate()
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&m))
}

// UnwrapEthereumMsg extracts MsgEthereumTx from wrapping sdk.Tx
func UnwrapEthereumMsg(tx *sdk.Tx, ethHash common.Hash) (*MsgEthereumTx, error) {
	if tx == nil {
		return nil, fmt.Errorf("invalid tx: nil")
	}

	for _, msg := range (*tx).GetMsgs() {
		ethMsg, ok := msg.(*MsgEthereumTx)
		if !ok {
			return nil, fmt.Errorf("invalid tx type: %T", tx)
		}
		txHash := ethMsg.AsTransaction().Hash()
		ethMsg.Hash = txHash.Hex()
		if txHash == ethHash {
			return ethMsg, nil
		}
	}

	return nil, fmt.Errorf("eth tx not found: %s", ethHash)
}

// DecodeTxResponse decodes an protobuf-encoded byte slice into TxResponse
func DecodeTxResponse(in []byte) (*MsgEthereumTxResponse, error) {
	var txMsgData sdk.TxMsgData
	if err := proto.Unmarshal(in, &txMsgData); err != nil {
		return nil, err
	}

	if len(txMsgData.MsgResponses) == 0 {
		return &MsgEthereumTxResponse{}, nil
	}

	var res MsgEthereumTxResponse
	if err := proto.Unmarshal(txMsgData.MsgResponses[0].Value, &res); err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to unmarshal tx response message data")
	}

	return &res, nil
}

var EmptyCodeHash = crypto.Keccak256(nil)

// BinSearch executes the binary search and hone in on an executable gas limit
func BinSearch(
	lo, hi uint64, executable func(uint64) (bool, *MsgEthereumTxResponse, error),
) (uint64, error) {
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)
		// If this errors, there was a consensus error, and the provided message
		// call or tx will never be accepted, regardless of how high we set the
		// gas limit.
		// Return the error directly, don't struggle anymore.
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi, nil
}

// GetSigners returns the expected signers for a MsgCreateFunToken message.
func (m MsgCreateFunToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgCreateFunToken) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender addr")
	}

	emptyBankDenom := m.FromBankDenom == ""
	emptyErc20 := m.FromErc20 == nil || m.FromErc20.Size() == 0

	if emptyErc20 && emptyBankDenom {
		return fmt.Errorf("either the \"from_erc20\" or \"from_bank_denom\" must be set")
	}

	if !emptyErc20 && !emptyBankDenom {
		return fmt.Errorf("either the \"from_erc20\" or \"from_bank_denom\" must be set (but not both)")
	}

	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgCreateFunToken) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgConvertCoinToEvm message.
func (m MsgConvertCoinToEvm) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgConvertCoinToEvm) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender addr")
	}
	if m.ToEthAddr.String() == "" || m.ToEthAddr.Size() == 0 {
		return fmt.Errorf("empty to_eth_addr")
	}
	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgConvertCoinToEvm) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&m))
}
