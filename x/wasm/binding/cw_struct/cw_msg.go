package cw_struct

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BindingMsg corresponds to the 'ExecuteMsg' enum in the CosmWasm binding
// contracts (Rust). It specifies which wasm execute messages can be called with
// Nibiru bindings and specifies the JSON schema that connects app ⇔ Wasm.
//
// See:
// - https://github.com/NibiruChain/cw-nibiru/blob/90df123f8d32d47b5b280ec6ae7dde0f9dbf2787/contracts/bindings-perp/src/msg.rs
type BindingMsg struct {
	// bindings-perp ExecuteMsg enum types
	OpenPosition  *OpenPosition  `json:"open_position,omitempty"`
	ClosePosition *ClosePosition `json:"close_position,omitempty"`
	//MultiLiquidate        *MultiLiquidate        `json:"multi_liquidate,omitempty"` // TODO
	AddMargin             *AddMargin             `json:"add_margin,omitempty"`
	RemoveMargin          *RemoveMargin          `json:"remove_margin,omitempty"`
	DonateToInsuranceFund *DonateToInsuranceFund `json:"donate_to_insurance_fund,omitempty"` // TODO
	InsuranceFundWithdraw *InsuranceFundWithdraw `json:"insurance_fund_withdraw,omitempty"`
	PegShift              *PegShift              `json:"peg_shift,omitempty"`
	DepthShift            *DepthShift            `json:"depth_shift,omitempty"`
	SetMarketEnabled      *SetMarketEnabled      `json:"set_market_enabled,omitempty"`
	CreateMarket          *CreateMarket          `json:"create_market,omitempty"`

	EditOracleParams *EditOracleParams `json:"edit_oracle_params,omitempty"`
}

type OpenPosition struct {
	Sender          string  `json:"sender"`
	Pair            string  `json:"pair"`
	IsLong          bool    `json:"is_long"`
	QuoteAmount     sdk.Int `json:"quote_amount"`
	Leverage        sdk.Dec `json:"leverage"`
	BaseAmountLimit sdk.Int `json:"base_amount_limit"`
}

type ClosePosition struct {
	Sender string `json:"sender"`
	Pair   string `json:"pair"`
}

type MultiLiquidate struct {
	Liquidations []LiquidationArgs `json:"liquidations"`
}

type LiquidationArgs struct {
	Pair   string `json:"pair"`
	Trader string `json:"trader"`
}

type AddMargin struct {
	Sender string   `json:"sender"`
	Pair   string   `json:"pair"`
	Margin sdk.Coin `json:"margin"`
}

type RemoveMargin struct {
	Sender string   `json:"sender"`
	Pair   string   `json:"pair"`
	Margin sdk.Coin `json:"margin"`
}

type PegShift struct {
	Pair    string  `json:"pair"`
	PegMult sdk.Dec `json:"peg_mult"`
}

type DepthShift struct {
	Pair      string  `json:"pair"`
	DepthMult sdk.Dec `json:"depth_mult"`
}

type DonateToInsuranceFund struct {
	Sender   string   `json:"sender"`
	Donation sdk.Coin `json:"donation"`
}

type EditOracleParams struct {
	VotePeriod         *sdk.Int `json:"vote_period,omitempty"`
	VoteThreshold      *sdk.Dec `json:"vote_threshold,omitempty"`
	RewardBand         *sdk.Dec `json:"reward_band,omitempty"`
	Whitelist          []string `json:"whitelist,omitempty"`
	SlashFraction      *sdk.Dec `json:"slash_fraction,omitempty"`
	SlashWindow        *sdk.Int `json:"slash_window,omitempty"`
	MinValidPerWindow  *sdk.Dec `json:"min_valid_per_window,omitempty"`
	TwapLookbackWindow *sdk.Int `json:"twap_lookback_window,omitempty"`
	MinVoters          *sdk.Int `json:"min_voters,omitempty"`
	ValidatorFeeRatio  *sdk.Dec `json:"validator_fee_ratio,omitempty"`
}

type InsuranceFundWithdraw struct {
	Amount sdk.Int `json:"amount"`
	To     string  `json:"to"`
}

type SetMarketEnabled struct {
	Pair    string `json:"pair"`
	Enabled bool   `json:"enabled"`
}

type CreateMarket struct {
	Pair         string        `json:"pair"`
	PegMult      sdk.Dec       `json:"peg_mult,omitempty"`
	SqrtDepth    sdk.Dec       `json:"sqrt_depth,omitempty"`
	MarketParams *MarketParams `json:"market_params,omitempty"`
}

type MarketParams struct {
	Pair    string
	Enabled bool `json:"enabled,omitempty"`
	// percentage that a single open or close position can alter the reserve
	// amounts
	PriceFluctuationLimitRatio sdk.Dec
	// the minimum margin ratio which a user must maintain on this market
	MaintenanceMarginRatio sdk.Dec `protobuf:"bytes,4,opt,name=maintenance_margin_ratio,json=maintenanceMarginRatio,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"maintenance_margin_ratio"`
	// the maximum leverage a user is able to be taken on this market
	MaxLeverage sdk.Dec `protobuf:"bytes,5,opt,name=max_leverage,json=maxLeverage,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"max_leverage"`
	// Latest cumulative premium fraction for a given pair.
	// Calculated once per funding rate interval.
	// A premium fraction is the difference between mark and index, divided by the
	// number of payments per day. (mark - index) / # payments in a day
	LatestCumulativePremiumFraction sdk.Dec `protobuf:"bytes,6,opt,name=latest_cumulative_premium_fraction,json=latestCumulativePremiumFraction,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"latest_cumulative_premium_fraction"`
	// the percentage of the notional given to the exchange when trading
	ExchangeFeeRatio sdk.Dec `protobuf:"bytes,7,opt,name=exchange_fee_ratio,json=exchangeFeeRatio,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"exchange_fee_ratio"`
	// the percentage of the notional transferred to the ecosystem fund when
	// trading
	EcosystemFundFeeRatio sdk.Dec `protobuf:"bytes,8,opt,name=ecosystem_fund_fee_ratio,json=ecosystemFundFeeRatio,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"ecosystem_fund_fee_ratio"`
	// the percentage of liquidated position that will be
	// given to out as a reward. Half of the liquidation fee is given to the
	// liquidator, and the other half is given to the ecosystem fund.
	LiquidationFeeRatio sdk.Dec `protobuf:"bytes,9,opt,name=liquidation_fee_ratio,json=liquidationFeeRatio,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"liquidation_fee_ratio"`
	// the portion of the position size we try to liquidate if the available
	// margin is higher than liquidation fee
	PartialLiquidationRatio sdk.Dec `protobuf:"bytes,10,opt,name=partial_liquidation_ratio,json=partialLiquidationRatio,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"partial_liquidation_ratio"`
	// specifies the interval on which the funding rate is updated
	FundingRateEpochId string `protobuf:"bytes,11,opt,name=funding_rate_epoch_id,json=fundingRateEpochId,proto3" json:"funding_rate_epoch_id,omitempty"`
	// amount of time to look back for TWAP calculations
	TwapLookbackWindow sdk.Int `protobuf:"bytes,12,opt,name=twap_lookback_window,json=twapLookbackWindow,proto3,stdduration" json:"twap_lookback_window"`
}
