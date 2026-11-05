// @generated
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventEpochStart {
    /// Epoch number, starting from 1.
    #[prost(uint64, tag="1")]
    pub epoch_number: u64,
    /// The start timestamp of the epoch.
    #[prost(message, optional, tag="2")]
    pub epoch_start_time: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventEpochEnd {
    /// Epoch number, starting from 1.
    #[prost(uint64, tag="1")]
    pub epoch_number: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EpochInfo {
    /// A string identifier for the epoch. e.g. "15min" or "1hour"
    #[prost(string, tag="1")]
    pub identifier: ::prost::alloc::string::String,
    /// When the epoch repetitino should start.
    #[prost(message, optional, tag="2")]
    pub start_time: ::core::option::Option<::prost_types::Timestamp>,
    /// How long each epoch lasts for.
    #[prost(message, optional, tag="3")]
    pub duration: ::core::option::Option<::prost_types::Duration>,
    /// The current epoch number, starting from 1.
    #[prost(uint64, tag="4")]
    pub current_epoch: u64,
    /// The start timestamp of the current epoch.
    #[prost(message, optional, tag="5")]
    pub current_epoch_start_time: ::core::option::Option<::prost_types::Timestamp>,
    /// Whether or not this epoch has started. Set to true if current blocktime >=
    /// start_time.
    #[prost(bool, tag="6")]
    pub epoch_counting_started: bool,
    /// The block height at which the current epoch started at.
    #[prost(int64, tag="7")]
    pub current_epoch_start_height: i64,
}
/// GenesisState defines the epochs module's genesis state.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, repeated, tag="1")]
    pub epochs: ::prost::alloc::vec::Vec<EpochInfo>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochInfosRequest {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochInfosResponse {
    #[prost(message, repeated, tag="1")]
    pub epochs: ::prost::alloc::vec::Vec<EpochInfo>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCurrentEpochRequest {
    #[prost(string, tag="1")]
    pub identifier: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCurrentEpochResponse {
    #[prost(uint64, tag="1")]
    pub current_epoch: u64,
}
// @@protoc_insertion_point(module)
