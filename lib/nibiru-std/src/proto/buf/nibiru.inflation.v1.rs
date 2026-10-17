// @generated
/// EventInflationDistribution: Emitted when NIBI tokens are minted on the
/// network based on Nibiru's inflation schedule.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventInflationDistribution {
    #[prost(message, optional, tag="1")]
    pub staking_rewards: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(message, optional, tag="2")]
    pub strategic_reserve: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(message, optional, tag="3")]
    pub community_pool: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// InflationDistribution defines the distribution in which inflation is
/// allocated through minting on each epoch (staking, community, strategic). It
/// excludes the team vesting distribution.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct InflationDistribution {
    /// staking_rewards defines the proportion of the minted_denom that is
    /// to be allocated as staking rewards
    #[prost(string, tag="1")]
    pub staking_rewards: ::prost::alloc::string::String,
    /// community_pool defines the proportion of the minted_denom that is to
    /// be allocated to the community pool
    #[prost(string, tag="2")]
    pub community_pool: ::prost::alloc::string::String,
    /// strategic_reserves defines the proportion of the minted_denom that
    /// is to be allocated to the strategic reserves module address
    #[prost(string, tag="3")]
    pub strategic_reserves: ::prost::alloc::string::String,
}
/// GenesisState defines the inflation module's genesis state.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// params defines all the parameters of the module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
    /// period is the amount of past periods, based on the epochs per period param
    #[prost(uint64, tag="2")]
    pub period: u64,
    /// skipped_epochs is the number of epochs that have passed while inflation is
    /// disabled
    #[prost(uint64, tag="3")]
    pub skipped_epochs: u64,
}
/// Params holds parameters for the inflation module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// inflation_enabled is the parameter that enables inflation and halts
    /// increasing the skipped_epochs
    #[prost(bool, tag="1")]
    pub inflation_enabled: bool,
    /// polynomial_factors takes in the variables to calculate polynomial
    /// inflation
    #[prost(string, repeated, tag="2")]
    pub polynomial_factors: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// inflation_distribution of the minted denom
    #[prost(message, optional, tag="3")]
    pub inflation_distribution: ::core::option::Option<InflationDistribution>,
    /// epochs_per_period is the number of epochs that must pass before a new
    /// period is created
    #[prost(uint64, tag="4")]
    pub epochs_per_period: u64,
    /// periods_per_year is the number of periods that occur in a year
    #[prost(uint64, tag="5")]
    pub periods_per_year: u64,
    /// max_period is the maximum number of periods that have inflation being 
    /// paid off. After this period, inflation will be disabled.
    #[prost(uint64, tag="6")]
    pub max_period: u64,
    /// has_inflation_started is the parameter that indicates if inflation has
    /// started. It's set to false at the starts, and stays at true when we toggle
    /// inflation on. It's used to track num skipped epochs
    #[prost(bool, tag="7")]
    pub has_inflation_started: bool,
}
/// QueryPeriodRequest is the request type for the Query/Period RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPeriodRequest {
}
/// QueryPeriodResponse is the response type for the Query/Period RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPeriodResponse {
    /// period is the current minting per epoch provision value.
    #[prost(uint64, tag="1")]
    pub period: u64,
}
/// QueryEpochMintProvisionRequest is the request type for the
/// Query/EpochMintProvision RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochMintProvisionRequest {
}
/// QueryEpochMintProvisionResponse is the response type for the
/// Query/EpochMintProvision RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochMintProvisionResponse {
    /// epoch_mint_provision is the current minting per epoch provision value.
    #[prost(message, optional, tag="1")]
    pub epoch_mint_provision: ::core::option::Option<crate::proto::cosmos::base::v1beta1::DecCoin>,
}
/// QuerySkippedEpochsRequest is the request type for the Query/SkippedEpochs RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySkippedEpochsRequest {
}
/// QuerySkippedEpochsResponse is the response type for the Query/SkippedEpochs
/// RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySkippedEpochsResponse {
    /// skipped_epochs is the number of epochs that the inflation module has been
    /// disabled.
    #[prost(uint64, tag="1")]
    pub skipped_epochs: u64,
}
/// QueryCirculatingSupplyRequest is the request type for the
/// Query/CirculatingSupply RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCirculatingSupplyRequest {
}
/// QueryCirculatingSupplyResponse is the response type for the
/// Query/CirculatingSupply RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCirculatingSupplyResponse {
    /// circulating_supply is the total amount of coins in circulation
    #[prost(message, optional, tag="1")]
    pub circulating_supply: ::core::option::Option<crate::proto::cosmos::base::v1beta1::DecCoin>,
}
/// QueryInflationRateRequest is the request type for the Query/InflationRate RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryInflationRateRequest {
}
/// QueryInflationRateResponse is the response type for the Query/InflationRate
/// RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryInflationRateResponse {
    /// inflation_rate by which the total supply increases within one period
    #[prost(string, tag="1")]
    pub inflation_rate: ::prost::alloc::string::String,
}
/// QueryParamsRequest is the request type for the Query/Params RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsRequest {
}
/// QueryParamsResponse is the response type for the Query/Params RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsResponse {
    /// params defines the parameters of the module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
/// MsgToggleInflation defines a message to enable or disable inflation.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgToggleInflation {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(bool, tag="2")]
    pub enable: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditInflationParams {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(bool, tag="2")]
    pub inflation_enabled: bool,
    #[prost(string, repeated, tag="3")]
    pub polynomial_factors: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    #[prost(message, optional, tag="4")]
    pub inflation_distribution: ::core::option::Option<InflationDistribution>,
    #[prost(string, tag="5")]
    pub epochs_per_period: ::prost::alloc::string::String,
    #[prost(string, tag="6")]
    pub periods_per_year: ::prost::alloc::string::String,
    #[prost(string, tag="7")]
    pub max_period: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgToggleInflationResponse {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditInflationParamsResponse {
}
/// MsgBurn: allows burning of any token
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgBurn {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub coin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgBurnResponse {
}
// @@protoc_insertion_point(module)
