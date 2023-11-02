package types

import (
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ensure Msg interface compliance at compile time
var (
	_ sdk.Msg = &MsgDelegateFeedConsent{}
	_ sdk.Msg = &MsgAggregateExchangeRatePrevote{}
	_ sdk.Msg = &MsgAggregateExchangeRateVote{}
	_ sdk.Msg = &MsgEditOracleParams{}
)

// oracle message types
const (
	TypeMsgDelegateFeedConsent          = "delegate_feeder"
	TypeMsgEditOracleParams             = "edit_oracle_params"
	TypeMsgAggregateExchangeRatePrevote = "aggregate_exchange_rate_prevote"
	TypeMsgAggregateExchangeRateVote    = "aggregate_exchange_rate_vote"
)

//-------------------------------------------------
//-------------------------------------------------

// NewMsgAggregateExchangeRatePrevote returns MsgAggregateExchangeRatePrevote instance
func NewMsgAggregateExchangeRatePrevote(hash AggregateVoteHash, feeder sdk.AccAddress, validator sdk.ValAddress) *MsgAggregateExchangeRatePrevote {
	return &MsgAggregateExchangeRatePrevote{
		Hash:      hash.String(),
		Feeder:    feeder.String(),
		Validator: validator.String(),
	}
}

// Route implements sdk.Msg
func (msg MsgAggregateExchangeRatePrevote) Route() string { return RouterKey }

// Type implements sdk.Msg
func (msg MsgAggregateExchangeRatePrevote) Type() string { return TypeMsgAggregateExchangeRatePrevote }

// GetSignBytes implements sdk.Msg
func (msg MsgAggregateExchangeRatePrevote) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgAggregateExchangeRatePrevote) GetSigners() []sdk.AccAddress {
	feeder, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{feeder}
}

// ValidateBasic Implements sdk.Msg
func (msg MsgAggregateExchangeRatePrevote) ValidateBasic() error {
	_, err := AggregateVoteHashFromHexString(msg.Hash)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidHash, "Invalid vote hash (%s)", err)
	}

	// HEX encoding doubles the hash length
	if len(msg.Hash) != tmhash.TruncatedSize*2 {
		return ErrInvalidHashLength
	}

	_, err = sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid feeder address (%s)", err)
	}

	_, err = sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid operator address (%s)", err)
	}

	return nil
}

// NewMsgAggregateExchangeRateVote returns MsgAggregateExchangeRateVote instance
// TODO(mercilex): accept ExchangeRatesTuples
func NewMsgAggregateExchangeRateVote(salt string, exchangeRates string, feeder sdk.AccAddress, validator sdk.ValAddress) *MsgAggregateExchangeRateVote {
	return &MsgAggregateExchangeRateVote{
		Salt:          salt,
		ExchangeRates: exchangeRates,
		Feeder:        feeder.String(),
		Validator:     validator.String(),
	}
}

// Route implements sdk.Msg
func (msg MsgAggregateExchangeRateVote) Route() string { return RouterKey }

// Type implements sdk.Msg
func (msg MsgAggregateExchangeRateVote) Type() string { return TypeMsgAggregateExchangeRateVote }

// GetSignBytes implements sdk.Msg
func (msg MsgAggregateExchangeRateVote) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgAggregateExchangeRateVote) GetSigners() []sdk.AccAddress {
	feeder, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{feeder}
}

// ValidateBasic implements sdk.Msg
func (msg MsgAggregateExchangeRateVote) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid feeder address (%s)", err)
	}

	_, err = sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid operator address (%s)", err)
	}

	if l := len(msg.ExchangeRates); l == 0 {
		return sdkerrors.Wrap(errors.ErrUnknownRequest, "must provide at least one oracle exchange rate")
	} else if l > 4096 {
		return sdkerrors.Wrap(errors.ErrInvalidRequest, "exchange rates string can not exceed 4096 characters")
	}

	exchangeRates, err := ParseExchangeRateTuples(msg.ExchangeRates)
	if err != nil {
		return sdkerrors.Wrap(errors.ErrInvalidCoins, "failed to parse exchange rates string cause: "+err.Error())
	}

	for _, exchangeRate := range exchangeRates {
		// Check overflow bit length
		if exchangeRate.ExchangeRate.BigInt().BitLen() > 255+sdk.DecimalPrecisionBits {
			return sdkerrors.Wrap(ErrInvalidExchangeRate, "overflow")
		}
	}

	if len(msg.Salt) > 4 || len(msg.Salt) < 1 {
		return sdkerrors.Wrap(ErrInvalidSaltLength, "salt length must be [1, 4]")
	}

	return nil
}

// NewMsgDelegateFeedConsent creates a MsgDelegateFeedConsent instance
func NewMsgDelegateFeedConsent(operatorAddress sdk.ValAddress, feederAddress sdk.AccAddress) *MsgDelegateFeedConsent {
	return &MsgDelegateFeedConsent{
		Operator: operatorAddress.String(),
		Delegate: feederAddress.String(),
	}
}

// Route implements sdk.Msg
func (msg MsgDelegateFeedConsent) Route() string { return RouterKey }

// Type implements sdk.Msg
func (msg MsgDelegateFeedConsent) Type() string { return TypeMsgDelegateFeedConsent }

// GetSignBytes implements sdk.Msg
func (msg MsgDelegateFeedConsent) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgDelegateFeedConsent) GetSigners() []sdk.AccAddress {
	operator, err := sdk.ValAddressFromBech32(msg.Operator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{sdk.AccAddress(operator)}
}

// ValidateBasic implements sdk.Msg
func (msg MsgDelegateFeedConsent) ValidateBasic() error {
	_, err := sdk.ValAddressFromBech32(msg.Operator)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid operator address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Delegate)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid delegate address (%s)", err)
	}

	return nil
}

func (msg MsgEditOracleParams) Route() string { return RouterKey }

func (msg MsgEditOracleParams) Type() string { return TypeMsgEditOracleParams }

// GetSignBytes implements sdk.Msg
func (msg MsgEditOracleParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgEditOracleParams) GetSigners() []sdk.AccAddress {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{sdk.AccAddress(sender)}
}

// ValidateBasic implements sdk.Msg
func (msg MsgEditOracleParams) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if msg.Params == nil {
		return sdkerrors.Wrap(errors.ErrInvalidRequest, "params cannot be nil")
	}

	return nil
}
