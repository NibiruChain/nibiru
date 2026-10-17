// @generated
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Market {
    /// the trading pair represented by this market
    /// always BASE:QUOTE, e.g. BTC:NUSD or ETH:NUSD
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    /// whether or not the market is enabled
    #[prost(bool, tag="2")]
    pub enabled: bool,
    /// the version of the Market, only one market can exist per pair, when one is closed it cannot be reactivated,
    /// so a new market must be created, this is the version of the market
    #[prost(uint64, tag="14")]
    pub version: u64,
    /// the minimum margin ratio which a user must maintain on this market
    #[prost(string, tag="3")]
    pub maintenance_margin_ratio: ::prost::alloc::string::String,
    /// the maximum leverage a user is able to be taken on this market
    #[prost(string, tag="4")]
    pub max_leverage: ::prost::alloc::string::String,
    /// Latest cumulative premium fraction for a given pair.
    /// Calculated once per funding rate interval.
    /// A premium fraction is the difference between mark and index, divided by the
    /// number of payments per day. (mark - index) / # payments in a day
    #[prost(string, tag="5")]
    pub latest_cumulative_premium_fraction: ::prost::alloc::string::String,
    /// the percentage of the notional given to the exchange when trading
    #[prost(string, tag="6")]
    pub exchange_fee_ratio: ::prost::alloc::string::String,
    /// the percentage of the notional transferred to the ecosystem fund when
    /// trading
    #[prost(string, tag="7")]
    pub ecosystem_fund_fee_ratio: ::prost::alloc::string::String,
    /// the percentage of liquidated position that will be
    /// given to out as a reward. Half of the liquidation fee is given to the
    /// liquidator, and the other half is given to the ecosystem fund.
    #[prost(string, tag="8")]
    pub liquidation_fee_ratio: ::prost::alloc::string::String,
    /// the portion of the position size we try to liquidate if the available
    /// margin is higher than liquidation fee
    #[prost(string, tag="9")]
    pub partial_liquidation_ratio: ::prost::alloc::string::String,
    /// specifies the interval on which the funding rate is updated
    #[prost(string, tag="10")]
    pub funding_rate_epoch_id: ::prost::alloc::string::String,
    /// amount of time to look back for TWAP calculations
    #[prost(message, optional, tag="11")]
    pub twap_lookback_window: ::core::option::Option<::prost_types::Duration>,
    /// the amount of collateral already credited from the ecosystem fund
    #[prost(message, optional, tag="12")]
    pub prepaid_bad_debt: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the maximum funding rate payment per epoch, this represents the maximum
    /// amount of funding that can be paid out per epoch as a percentage of the
    /// position size
    #[prost(string, tag="13")]
    pub max_funding_rate: ::prost::alloc::string::String,
    /// the pair of the oracle that is used to determine the index price
    /// for the market
    #[prost(string, tag="15")]
    pub oracle_pair: ::prost::alloc::string::String,
}
/// MarketLastVersion is used to store the last version of the market
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MarketLastVersion {
    /// version of the market
    #[prost(uint64, tag="1")]
    pub version: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Amm {
    /// identifies the market this AMM belongs to
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    /// the version of the AMM, only one AMM can exist per pair, when one is closed it cannot be reactivated,
    /// so a new AMM must be created, this is the version of the AMM
    #[prost(uint64, tag="8")]
    pub version: u64,
    /// the amount of base reserves this AMM has
    #[prost(string, tag="2")]
    pub base_reserve: ::prost::alloc::string::String,
    /// the amount of quote reserves this AMM has
    #[prost(string, tag="3")]
    pub quote_reserve: ::prost::alloc::string::String,
    /// sqrt(k)
    #[prost(string, tag="4")]
    pub sqrt_depth: ::prost::alloc::string::String,
    /// the price multiplier of the dynamic AMM
    #[prost(string, tag="5")]
    pub price_multiplier: ::prost::alloc::string::String,
    /// Total long refers to the sum of long open notional in base.
    #[prost(string, tag="6")]
    pub total_long: ::prost::alloc::string::String,
    /// Total short refers to the sum of short open notional in base.
    #[prost(string, tag="7")]
    pub total_short: ::prost::alloc::string::String,
    /// The settlement price if the AMM is settled.
    #[prost(string, tag="9")]
    pub settlement_price: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Position {
    /// address identifies the address owner of this position
    #[prost(string, tag="1")]
    pub trader_address: ::prost::alloc::string::String,
    /// pair identifies the pair associated with this position
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    /// the position size
    #[prost(string, tag="3")]
    pub size: ::prost::alloc::string::String,
    /// amount of margin remaining in the position
    #[prost(string, tag="4")]
    pub margin: ::prost::alloc::string::String,
    /// value of position in quote assets when opened
    #[prost(string, tag="5")]
    pub open_notional: ::prost::alloc::string::String,
    /// The most recent cumulative premium fraction this position has.
    /// Used to calculate the next funding payment.
    #[prost(string, tag="6")]
    pub latest_cumulative_premium_fraction: ::prost::alloc::string::String,
    /// last block number this position was updated
    #[prost(int64, tag="7")]
    pub last_updated_block_number: i64,
}
/// a snapshot of the perp.amm's reserves at a given point in time
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ReserveSnapshot {
    #[prost(message, optional, tag="1")]
    pub amm: ::core::option::Option<Amm>,
    /// milliseconds since unix epoch
    #[prost(int64, tag="2")]
    pub timestamp_ms: i64,
}
/// DNRAllocation represents a rebates allocation for a given epoch.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct DnrAllocation {
    /// epoch defines the reference epoch for the allocation.
    #[prost(uint64, tag="1")]
    pub epoch: u64,
    /// amount of DNR allocated for the epoch.
    #[prost(message, repeated, tag="2")]
    pub amount: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// The direction that the user is trading in
/// LONG means the user is going long the base asset (e.g. buy BTC)
/// SHORT means the user is shorting the base asset (e.g. sell BTC)
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum Direction {
    Unspecified = 0,
    Long = 1,
    Short = 2,
}
impl Direction {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            Direction::Unspecified => "DIRECTION_UNSPECIFIED",
            Direction::Long => "LONG",
            Direction::Short => "SHORT",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "DIRECTION_UNSPECIFIED" => Some(Self::Unspecified),
            "LONG" => Some(Self::Long),
            "SHORT" => Some(Self::Short),
            _ => None,
        }
    }
}
/// Enumerates different options of calculating twap.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum TwapCalcOption {
    Unspecified = 0,
    /// Spot price from quote asset reserve / base asset reserve
    Spot = 1,
    /// Swapping with quote assets, output denominated in base assets
    QuoteAssetSwap = 2,
    /// Swapping with base assets, output denominated in quote assets
    BaseAssetSwap = 3,
}
impl TwapCalcOption {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            TwapCalcOption::Unspecified => "TWAP_CALC_OPTION_UNSPECIFIED",
            TwapCalcOption::Spot => "SPOT",
            TwapCalcOption::QuoteAssetSwap => "QUOTE_ASSET_SWAP",
            TwapCalcOption::BaseAssetSwap => "BASE_ASSET_SWAP",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "TWAP_CALC_OPTION_UNSPECIFIED" => Some(Self::Unspecified),
            "SPOT" => Some(Self::Spot),
            "QUOTE_ASSET_SWAP" => Some(Self::QuoteAssetSwap),
            "BASE_ASSET_SWAP" => Some(Self::BaseAssetSwap),
            _ => None,
        }
    }
}
/// Emitted when a position changes.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PositionChangedEvent {
    #[prost(message, optional, tag="1")]
    pub final_position: ::core::option::Option<Position>,
    /// Position notional (in quote units) after the change. In general,
    /// 'notional = baseAmount * priceQuotePerBase', where size is the baseAmount.
    #[prost(string, tag="2")]
    pub position_notional: ::prost::alloc::string::String,
    /// Transaction fee paid. A "taker" fee.
    #[prost(message, optional, tag="3")]
    pub transaction_fee: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// realize profits and losses after the change
    #[prost(string, tag="4")]
    pub realized_pnl: ::prost::alloc::string::String,
    /// Amount of bad debt cleared by the PerpEF during the change.
    /// Bad debt is negative net margin past the liquidation point of a position.
    #[prost(message, optional, tag="5")]
    pub bad_debt: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// A funding payment made or received by the trader on the current position.
    /// 'fundingPayment' is positive if 'owner' is the sender and negative if 'owner'
    /// is the receiver of the payment. Its magnitude is abs(size * fundingRate).
    /// Funding payments act to converge the mark price and index price
    /// (average price on major exchanges).
    #[prost(string, tag="6")]
    pub funding_payment: ::prost::alloc::string::String,
    /// The block number at which this position was changed.
    #[prost(int64, tag="7")]
    pub block_height: i64,
    /// margin_to_user is the amount of collateral received by the trader during
    /// the position change. A positve value indicates that the trader received
    /// funds, while a negative value indicates that the trader spent funds.
    #[prost(string, tag="8")]
    pub margin_to_user: ::prost::alloc::string::String,
    /// change_reason describes the reason for why the position resulted in a
    /// change. Change type can take the following values:
    ///
    /// - CHANGE_REASON_UNSPECIFIED: Unspecified change reason.
    /// - CHANGE_REASON_ADD_MARGIN: Margin was added to the position.
    /// - CHANGE_REASON_REMOVE_MARGIN: Margin was removed from the position.
    /// - CHANGE_REASON_OPEN_POSITION: A new position was opened.
    /// - CHANGE_REASON_CLOSE_POSITION: An existing position was closed.
    #[prost(string, tag="9")]
    pub change_reason: ::prost::alloc::string::String,
    /// exchanged_size represent the change in size for an existing position
    /// after the change. A positive value indicates that the position size
    /// increased, while a negative value indicates that the position size
    /// decreased.
    #[prost(string, tag="10")]
    pub exchanged_size: ::prost::alloc::string::String,
    /// exchanged_notional represent the change in notional for an existing
    /// position after the change. A positive value indicates that the position
    /// notional increased, while a negative value indicates that the position
    /// notional decreased.
    #[prost(string, tag="11")]
    pub exchanged_notional: ::prost::alloc::string::String,
}
/// Emitted when a position is liquidated. Wraps a PositionChanged event since a
/// liquidation causes position changes.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PositionLiquidatedEvent {
    #[prost(message, optional, tag="1")]
    pub position_changed_event: ::core::option::Option<PositionChangedEvent>,
    /// Address of the account that executed the tx.
    #[prost(string, tag="2")]
    pub liquidator_address: ::prost::alloc::string::String,
    /// Commission (in margin units) received by 'liquidator'.
    #[prost(message, optional, tag="3")]
    pub fee_to_liquidator: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// Commission (in margin units) given to the ecosystem fund.
    #[prost(message, optional, tag="4")]
    pub fee_to_ecosystem_fund: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// Emitted when a position is settled.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PositionSettledEvent {
    /// Identifier for the virtual pool of the position.
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    /// Owner of the position.
    #[prost(string, tag="2")]
    pub trader_address: ::prost::alloc::string::String,
    /// Settled coin as dictated by the settlement price of the perp.amm.
    #[prost(message, repeated, tag="3")]
    pub settled_coins: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// Emitted when the funding rate changes for a market.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct FundingRateChangedEvent {
    /// The pair for which the funding rate was calculated.
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    /// The mark price of the pair.
    #[prost(string, tag="2")]
    pub mark_price_twap: ::prost::alloc::string::String,
    /// The oracle index price of the pair.
    #[prost(string, tag="3")]
    pub index_price_twap: ::prost::alloc::string::String,
    /// The latest premium fraction just calculated.
    #[prost(string, tag="5")]
    pub premium_fraction: ::prost::alloc::string::String,
    /// The market's latest cumulative premium fraction.
    /// The funding payment a position will pay is the difference between this
    /// value and the latest cumulative premium fraction on the position,
    /// multiplied by the position size.
    #[prost(string, tag="6")]
    pub cumulative_premium_fraction: ::prost::alloc::string::String,
}
/// Emitted when liquidation fails.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct LiquidationFailedEvent {
    /// The pair for which we are trying to liquidate.
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    /// owner of the position.
    #[prost(string, tag="2")]
    pub trader: ::prost::alloc::string::String,
    /// Address of the account that executed the tx.
    #[prost(string, tag="3")]
    pub liquidator: ::prost::alloc::string::String,
    /// Reason for the liquidation failure.
    #[prost(enumeration="liquidation_failed_event::LiquidationFailedReason", tag="4")]
    pub reason: i32,
}
/// Nested message and enum types in `LiquidationFailedEvent`.
pub mod liquidation_failed_event {
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
    #[repr(i32)]
    pub enum LiquidationFailedReason {
        Unspecified = 0,
        /// the position is healthy and does not need to be liquidated.
        PositionHealthy = 1,
        /// the pair does not exist.
        NonexistentPair = 2,
        /// the position does not exist.
        NonexistentPosition = 3,
    }
    impl LiquidationFailedReason {
        /// String value of the enum field names used in the ProtoBuf definition.
        ///
        /// The values are not transformed in any way and thus are considered stable
        /// (if the ProtoBuf definition does not change) and safe for programmatic use.
        pub fn as_str_name(&self) -> &'static str {
            match self {
                LiquidationFailedReason::Unspecified => "UNSPECIFIED",
                LiquidationFailedReason::PositionHealthy => "POSITION_HEALTHY",
                LiquidationFailedReason::NonexistentPair => "NONEXISTENT_PAIR",
                LiquidationFailedReason::NonexistentPosition => "NONEXISTENT_POSITION",
            }
        }
        /// Creates an enum from field names used in the ProtoBuf definition.
        pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
            match value {
                "UNSPECIFIED" => Some(Self::Unspecified),
                "POSITION_HEALTHY" => Some(Self::PositionHealthy),
                "NONEXISTENT_PAIR" => Some(Self::NonexistentPair),
                "NONEXISTENT_POSITION" => Some(Self::NonexistentPosition),
                _ => None,
            }
        }
    }
}
/// This event is emitted when the amm is updated, which can be triggered by
/// the following events:
///
/// - swap
/// - edit price multiplier
/// - edit depth
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AmmUpdatedEvent {
    /// the final state of the AMM
    #[prost(message, optional, tag="1")]
    pub final_amm: ::core::option::Option<Amm>,
    /// The mark price of the pair.
    #[prost(string, tag="2")]
    pub mark_price_twap: ::prost::alloc::string::String,
    /// The oracle index price of the pair.
    #[prost(string, tag="3")]
    pub index_price_twap: ::prost::alloc::string::String,
}
/// This event is emitted at the end of every block for persisting market changes
/// off-chain
///
/// Market changes are triggered by the following actions:
///
/// - disabling market
/// - changing market fees
/// - bad debt is prepaid by the ecosystem fund
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MarketUpdatedEvent {
    /// the final state of the market
    #[prost(message, optional, tag="1")]
    pub final_market: ::core::option::Option<Market>,
}
/// EventShiftPegMultiplier: ABCI event emitted from MsgShiftPegMultiplier
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventShiftPegMultiplier {
    #[prost(string, tag="1")]
    pub old_peg_multiplier: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub new_peg_multiplier: ::prost::alloc::string::String,
    #[prost(message, optional, tag="3")]
    pub cost_paid: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// EventShiftSwapInvariant: ABCI event emitted from MsgShiftSwapInvariant
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventShiftSwapInvariant {
    #[prost(string, tag="1")]
    pub old_swap_invariant: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub new_swap_invariant: ::prost::alloc::string::String,
    #[prost(message, optional, tag="3")]
    pub cost_paid: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// GenesisState defines the perp module's genesis state.
/// Thge genesis state is used not only to start the network but also useful for
/// exporting and importing state during network upgrades.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, repeated, tag="2")]
    pub markets: ::prost::alloc::vec::Vec<Market>,
    #[prost(message, repeated, tag="3")]
    pub amms: ::prost::alloc::vec::Vec<Amm>,
    #[prost(message, repeated, tag="4")]
    pub positions: ::prost::alloc::vec::Vec<GenesisPosition>,
    #[prost(message, repeated, tag="5")]
    pub reserve_snapshots: ::prost::alloc::vec::Vec<ReserveSnapshot>,
    #[prost(uint64, tag="6")]
    pub dnr_epoch: u64,
    /// For testing purposes, we allow the collateral to be set at genesis
    #[prost(string, tag="11")]
    pub collateral_denom: ::prost::alloc::string::String,
    #[prost(message, repeated, tag="7")]
    pub trader_volumes: ::prost::alloc::vec::Vec<genesis_state::TraderVolume>,
    #[prost(message, repeated, tag="8")]
    pub global_discount: ::prost::alloc::vec::Vec<genesis_state::Discount>,
    #[prost(message, repeated, tag="9")]
    pub custom_discounts: ::prost::alloc::vec::Vec<genesis_state::CustomDiscount>,
    #[prost(message, repeated, tag="10")]
    pub market_last_versions: ::prost::alloc::vec::Vec<GenesisMarketLastVersion>,
    #[prost(message, repeated, tag="13")]
    pub global_volumes: ::prost::alloc::vec::Vec<genesis_state::GlobalVolume>,
    #[prost(message, repeated, tag="12")]
    pub rebates_allocations: ::prost::alloc::vec::Vec<DnrAllocation>,
    #[prost(string, tag="14")]
    pub dnr_epoch_name: ::prost::alloc::string::String,
}
/// Nested message and enum types in `GenesisState`.
pub mod genesis_state {
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
    pub struct TraderVolume {
        #[prost(string, tag="1")]
        pub trader: ::prost::alloc::string::String,
        #[prost(uint64, tag="2")]
        pub epoch: u64,
        #[prost(string, tag="3")]
        pub volume: ::prost::alloc::string::String,
    }
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Discount {
        #[prost(string, tag="1")]
        pub fee: ::prost::alloc::string::String,
        #[prost(string, tag="2")]
        pub volume: ::prost::alloc::string::String,
    }
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
    pub struct CustomDiscount {
        #[prost(string, tag="1")]
        pub trader: ::prost::alloc::string::String,
        #[prost(message, optional, tag="2")]
        pub discount: ::core::option::Option<Discount>,
    }
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
    pub struct GlobalVolume {
        #[prost(uint64, tag="1")]
        pub epoch: u64,
        #[prost(string, tag="2")]
        pub volume: ::prost::alloc::string::String,
    }
}
/// GenesisMarketLastVersion is the last version including pair only used for
/// genesis
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisMarketLastVersion {
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub version: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisPosition {
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub version: u64,
    #[prost(message, optional, tag="3")]
    pub position: ::core::option::Option<Position>,
}
// ---------------------------------------- Positions

/// QueryPositionsRequest: Request type for the
/// "nibiru.perp.v2.Query/Positions" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPositionsRequest {
    #[prost(string, tag="1")]
    pub trader: ::prost::alloc::string::String,
}
/// QueryPositionsResponse: Response type for the
/// "nibiru.perp.v2.Query/Positions" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPositionsResponse {
    #[prost(message, repeated, tag="1")]
    pub positions: ::prost::alloc::vec::Vec<QueryPositionResponse>,
}
/// QueryPositionStoreRequest: Request type for the
/// "nibiru.perp.v2.Query/PositionStore" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPositionStoreRequest {
    /// pagination defines a paginated request
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// QueryPositionStoreResponse: Response type for the
/// "nibiru.perp.v2.Query/PositionStore" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPositionStoreResponse {
    /// Position responses: collection of all stored positions (with pagination)
    #[prost(message, repeated, tag="1")]
    pub positions: ::prost::alloc::vec::Vec<Position>,
    /// pagination defines a paginated response
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageResponse>,
}
// ---------------------------------------- Position

/// QueryPositionRequest: Request type for the
/// "nibiru.perp.v2.Query/Position" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPositionRequest {
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub trader: ::prost::alloc::string::String,
}
/// QueryPositionResponse: Response type for the
/// "nibiru.perp.v2.Query/Position" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPositionResponse {
    /// The position as it exists in the blockchain state
    #[prost(message, optional, tag="1")]
    pub position: ::core::option::Option<Position>,
    /// The position's current notional value, if it were to be entirely closed (in
    /// margin units).
    #[prost(string, tag="2")]
    pub position_notional: ::prost::alloc::string::String,
    /// The position's unrealized PnL.
    #[prost(string, tag="3")]
    pub unrealized_pnl: ::prost::alloc::string::String,
    /// margin ratio of the position based on the spot price
    #[prost(string, tag="4")]
    pub margin_ratio: ::prost::alloc::string::String,
}
// ---------------------------------------- QueryModuleAccounts

/// QueryModuleAccountsRequest: Request type for the
/// "nibiru.perp.v2.Query/ModuleAccounts" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryModuleAccountsRequest {
}
/// QueryModuleAccountsResponse: Response type for the
/// "nibiru.perp.v2.Query/ModuleAccounts" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryModuleAccountsResponse {
    #[prost(message, repeated, tag="1")]
    pub accounts: ::prost::alloc::vec::Vec<AccountWithBalance>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AccountWithBalance {
    #[prost(string, tag="1")]
    pub name: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub address: ::prost::alloc::string::String,
    #[prost(message, repeated, tag="3")]
    pub balance: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AmmMarket {
    #[prost(message, optional, tag="1")]
    pub market: ::core::option::Option<Market>,
    #[prost(message, optional, tag="2")]
    pub amm: ::core::option::Option<Amm>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryMarketsRequest {
    #[prost(bool, tag="1")]
    pub versioned: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryMarketsResponse {
    #[prost(message, repeated, tag="1")]
    pub amm_markets: ::prost::alloc::vec::Vec<AmmMarket>,
}
// ---------------------------------------- QueryCollateral

/// QueryCollateralRequest: Request type for the
/// "nibiru.perp.v2.Query/Collateral" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCollateralRequest {
}
/// QueryCollateralRequest: Response type for the
/// "nibiru.perp.v2.Query/Collateral" gRPC service method
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCollateralResponse {
    #[prost(string, tag="1")]
    pub collateral_denom: ::prost::alloc::string::String,
}
// -------------------------- Settle Position --------------------------

/// MsgSettlePosition: Msg to remove margin. 
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSettlePosition {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(uint64, tag="3")]
    pub version: u64,
}
// -------------------------- RemoveMargin --------------------------

/// MsgRemoveMargin: Msg to remove margin. 
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgRemoveMargin {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(message, optional, tag="3")]
    pub margin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgRemoveMarginResponse {
    /// tokens transferred back to the trader
    #[prost(message, optional, tag="1")]
    pub margin_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the funding payment applied on this position interaction
    #[prost(string, tag="2")]
    pub funding_payment: ::prost::alloc::string::String,
    /// The resulting position
    #[prost(message, optional, tag="3")]
    pub position: ::core::option::Option<Position>,
}
// -------------------------- AddMargin --------------------------

/// MsgAddMargin: Msg to remove margin. 
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAddMargin {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(message, optional, tag="3")]
    pub margin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAddMarginResponse {
    #[prost(string, tag="1")]
    pub funding_payment: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub position: ::core::option::Option<Position>,
}
// -------------------------- Liquidation --------------------------

#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgMultiLiquidate {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(message, repeated, tag="2")]
    pub liquidations: ::prost::alloc::vec::Vec<msg_multi_liquidate::Liquidation>,
}
/// Nested message and enum types in `MsgMultiLiquidate`.
pub mod msg_multi_liquidate {
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Liquidation {
        #[prost(string, tag="1")]
        pub pair: ::prost::alloc::string::String,
        #[prost(string, tag="2")]
        pub trader: ::prost::alloc::string::String,
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgMultiLiquidateResponse {
    #[prost(message, repeated, tag="1")]
    pub liquidations: ::prost::alloc::vec::Vec<msg_multi_liquidate_response::LiquidationResponse>,
}
/// Nested message and enum types in `MsgMultiLiquidateResponse`.
pub mod msg_multi_liquidate_response {
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
    pub struct LiquidationResponse {
        #[prost(bool, tag="1")]
        pub success: bool,
        #[prost(string, tag="2")]
        pub error: ::prost::alloc::string::String,
        /// nullable since no fee is taken on failed liquidation
        #[prost(message, optional, tag="3")]
        pub liquidator_fee: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
        /// perp ecosystem fund
        #[prost(message, optional, tag="4")]
        pub perp_ef_fee: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
        // nullable since no fee is taken on failed liquidation

        #[prost(string, tag="5")]
        pub trader: ::prost::alloc::string::String,
        #[prost(string, tag="6")]
        pub pair: ::prost::alloc::string::String,
    }
}
// -------------------------- MarketOrder --------------------------

#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgMarketOrder {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(enumeration="Direction", tag="3")]
    pub side: i32,
    #[prost(string, tag="4")]
    pub quote_asset_amount: ::prost::alloc::string::String,
    #[prost(string, tag="5")]
    pub leverage: ::prost::alloc::string::String,
    #[prost(string, tag="6")]
    pub base_asset_amount_limit: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgMarketOrderResponse {
    #[prost(message, optional, tag="1")]
    pub position: ::core::option::Option<Position>,
    /// The amount of quote assets exchanged.
    #[prost(string, tag="2")]
    pub exchanged_notional_value: ::prost::alloc::string::String,
    /// The amount of base assets exchanged.
    #[prost(string, tag="3")]
    pub exchanged_position_size: ::prost::alloc::string::String,
    /// The funding payment applied on this position change, measured in quote
    /// units.
    #[prost(string, tag="4")]
    pub funding_payment: ::prost::alloc::string::String,
    /// The amount of PnL realized on this position changed, measured in quote
    /// units.
    #[prost(string, tag="5")]
    pub realized_pnl: ::prost::alloc::string::String,
    /// The unrealized PnL in the position after the position change, measured in
    /// quote units.
    #[prost(string, tag="6")]
    pub unrealized_pnl_after: ::prost::alloc::string::String,
    /// The amount of margin the trader has to give to the vault.
    /// A negative value means the vault pays the trader.
    #[prost(string, tag="7")]
    pub margin_to_vault: ::prost::alloc::string::String,
    /// The position's notional value after the position change, measured in quote
    /// units.
    #[prost(string, tag="8")]
    pub position_notional: ::prost::alloc::string::String,
}
// -------------------------- ClosePosition --------------------------

#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgClosePosition {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgClosePositionResponse {
    /// The amount of quote assets exchanged.
    #[prost(string, tag="1")]
    pub exchanged_notional_value: ::prost::alloc::string::String,
    /// The amount of base assets exchanged.
    #[prost(string, tag="2")]
    pub exchanged_position_size: ::prost::alloc::string::String,
    /// The funding payment applied on this position change, measured in quote
    /// units.
    #[prost(string, tag="3")]
    pub funding_payment: ::prost::alloc::string::String,
    /// The amount of PnL realized on this position changed, measured in quote
    /// units.
    #[prost(string, tag="4")]
    pub realized_pnl: ::prost::alloc::string::String,
    /// The amount of margin the trader receives after closing the position, from
    /// the vault. Should never be negative.
    #[prost(string, tag="5")]
    pub margin_to_trader: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgPartialClose {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub size: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgPartialCloseResponse {
    /// The amount of quote assets exchanged.
    #[prost(string, tag="1")]
    pub exchanged_notional_value: ::prost::alloc::string::String,
    /// The amount of base assets exchanged.
    #[prost(string, tag="2")]
    pub exchanged_position_size: ::prost::alloc::string::String,
    /// The funding payment applied on this position change, measured in quote
    /// units.
    #[prost(string, tag="3")]
    pub funding_payment: ::prost::alloc::string::String,
    /// The amount of PnL realized on this position changed, measured in quote
    /// units.
    #[prost(string, tag="4")]
    pub realized_pnl: ::prost::alloc::string::String,
    /// The amount of margin the trader receives after closing the position, from
    /// the vault. Should never be negative.
    #[prost(string, tag="5")]
    pub margin_to_trader: ::prost::alloc::string::String,
}
// -------------------------- DonateToEcosystemFund --------------------------

#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgDonateToEcosystemFund {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// donation to the EF
    #[prost(message, optional, tag="2")]
    pub donation: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgDonateToEcosystemFundResponse {
}
// -----------------------  MsgChangeCollateralDenom -----------------------

/// MsgChangeCollateralDenom: Changes the collateral denom for the module.
/// \[SUDO\] Only callable by sudoers.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeCollateralDenom {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub new_denom: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeCollateralDenomResponse {
}
/// -------------------------- AllocateEpochRebates --------------------------
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAllocateEpochRebates {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// rebates to allocate
    #[prost(message, repeated, tag="2")]
    pub rebates: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAllocateEpochRebatesResponse {
    #[prost(message, repeated, tag="1")]
    pub total_epoch_rebates: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// -------------------------- WithdrawEpochRebates --------------------------
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgWithdrawEpochRebates {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(uint64, repeated, tag="2")]
    pub epochs: ::prost::alloc::vec::Vec<u64>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgWithdrawEpochRebatesResponse {
    #[prost(message, repeated, tag="1")]
    pub withdrawn_rebates: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
// -------------------------- ShiftPegMultiplier --------------------------

/// MsgShiftPegMultiplier: gRPC tx msg for changing the peg multiplier.
/// \[SUDO\] Only callable sudoers.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgShiftPegMultiplier {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub new_peg_mult: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgShiftPegMultiplierResponse {
}
// -------------------------- ShiftSwapInvariant --------------------------

/// MsgShiftSwapInvariant: gRPC tx msg for changing the swap invariant.
/// \[SUDO\] Only callable sudoers.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgShiftSwapInvariant {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub new_swap_invariant: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgShiftSwapInvariantResponse {
}
// -------------------------- WithdrawFromPerpFund --------------------------

/// MsgWithdrawFromPerpFund: gRPC tx msg for changing the swap invariant.
/// \[SUDO\] Only callable sudoers.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgWithdrawFromPerpFund {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub amount: ::prost::alloc::string::String,
    /// Optional denom in case withdrawing assets aside from NUSD.
    #[prost(string, tag="3")]
    pub denom: ::prost::alloc::string::String,
    #[prost(string, tag="4")]
    pub to_addr: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgWithdrawFromPerpFundResponse {
}
// -------------------------- CloseMarket --------------------------

/// CloseMarket: gRPC tx msg for closing a market.
/// Admin-only.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCloseMarket {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub pair: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCloseMarketResponse {
}
// @@protoc_insertion_point(module)
