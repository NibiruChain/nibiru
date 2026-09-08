// @generated
/// Params defines the module parameters for the x/oracle module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// VotePeriod defines the number of blocks during which voting takes place.
    #[prost(uint64, tag="1")]
    pub vote_period: u64,
    /// VoteThreshold specifies the minimum proportion of votes that must be
    /// received for a ballot to pass.
    #[prost(string, tag="2")]
    pub vote_threshold: ::prost::alloc::string::String,
    /// RewardBand defines a maximum divergence that a price vote can have from the
    /// weighted median in the ballot. If a vote lies within the valid range
    /// defined by:
    /// 	μ := weightedMedian,
    /// 	validRange := μ ± (μ * rewardBand / 2),
    /// then rewards are added to the validator performance.
    /// Note that if the reward band is smaller than 1 standard
    /// deviation, the band is taken to be 1 standard deviation.a price
    #[prost(string, tag="3")]
    pub reward_band: ::prost::alloc::string::String,
    /// The set of whitelisted markets, or asset pairs, for the module.
    /// Ex. '\["unibi:uusd","ubtc:uusd"\]'
    #[prost(string, repeated, tag="4")]
    pub whitelist: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// SlashFraction returns the proportion of an oracle's stake that gets
    /// slashed in the event of slashing. `SlashFraction` specifies the exact
    /// penalty for failing a voting period.
    #[prost(string, tag="5")]
    pub slash_fraction: ::prost::alloc::string::String,
    /// SlashWindow returns the number of voting periods that specify a
    /// "slash window". After each slash window, all oracles that have missed more
    /// than the penalty threshold are slashed. Missing the penalty threshold is
    /// synonymous with submitting fewer valid votes than `MinValidPerWindow`.
    #[prost(uint64, tag="6")]
    pub slash_window: u64,
    #[prost(string, tag="7")]
    pub min_valid_per_window: ::prost::alloc::string::String,
    /// Amount of time to look back for TWAP calculations.
    /// Ex: "900.000000069s" corresponds to 900 seconds and 69 nanoseconds in JSON.
    #[prost(message, optional, tag="8")]
    pub twap_lookback_window: ::core::option::Option<::prost_types::Duration>,
    /// The minimum number of voters (i.e. oracle validators) per pair for it to be
    /// considered a passing ballot. Recommended at least 4.
    #[prost(uint64, tag="9")]
    pub min_voters: u64,
    /// The validator fee ratio that is given to validators every epoch.
    #[prost(string, tag="10")]
    pub validator_fee_ratio: ::prost::alloc::string::String,
    #[prost(uint64, tag="11")]
    pub expiration_blocks: u64,
}
/// Struct for aggregate prevoting on the ExchangeRateVote.
/// The purpose of aggregate prevote is to hide vote exchange rates with hash
/// which is formatted as hex string in
/// SHA256("{salt}:({pair},{exchange_rate})|...|({pair},{exchange_rate}):{voter}")
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AggregateExchangeRatePrevote {
    #[prost(string, tag="1")]
    pub hash: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub voter: ::prost::alloc::string::String,
    #[prost(uint64, tag="3")]
    pub submit_block: u64,
}
/// MsgAggregateExchangeRateVote - struct for voting on
/// the exchange rates different assets.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AggregateExchangeRateVote {
    #[prost(message, repeated, tag="1")]
    pub exchange_rate_tuples: ::prost::alloc::vec::Vec<ExchangeRateTuple>,
    #[prost(string, tag="2")]
    pub voter: ::prost::alloc::string::String,
}
/// ExchangeRateTuple - struct to store interpreted exchange rates data to store
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ExchangeRateTuple {
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub exchange_rate: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ExchangeRateAtBlock {
    #[prost(string, tag="1")]
    pub exchange_rate: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub created_block: u64,
    /// Block timestamp for the block where the oracle came to consensus for this
    /// price. This timestamp is a conventional Unix millisecond time, i.e. the
    /// number of milliseconds elapsed since January 1, 1970 UTC. 
    #[prost(int64, tag="3")]
    pub block_timestamp_ms: i64,
}
/// Rewards defines a credit object towards validators
/// which provide prices faithfully for different pairs.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Rewards {
    /// id uniquely identifies the rewards instance of the pair
    #[prost(uint64, tag="1")]
    pub id: u64,
    /// vote_periods defines the vote periods left in which rewards will be
    /// distributed.
    #[prost(uint64, tag="2")]
    pub vote_periods: u64,
    /// Coins defines the amount of coins to distribute in a single vote period.
    #[prost(message, repeated, tag="3")]
    pub coins: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// Emitted when a price is posted
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventPriceUpdate {
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub price: ::prost::alloc::string::String,
    #[prost(int64, tag="3")]
    pub timestamp_ms: i64,
}
/// Emitted when a valoper delegates oracle voting rights to a feeder address.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventDelegateFeederConsent {
    /// Validator is the Bech32 address that is delegating voting rights.
    #[prost(string, tag="1")]
    pub validator: ::prost::alloc::string::String,
    /// Feeder is the delegate or representative that will be able to send
    /// vote and prevote transaction messages.
    #[prost(string, tag="2")]
    pub feeder: ::prost::alloc::string::String,
}
/// Emitted by MsgAggregateExchangeVote when an aggregate vote is added to state
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventAggregateVote {
    /// Validator is the Bech32 address to which the vote will be credited.
    #[prost(string, tag="1")]
    pub validator: ::prost::alloc::string::String,
    /// Feeder is the delegate or representative that will send vote and prevote
    /// transaction messages on behalf of the voting validator.
    #[prost(string, tag="2")]
    pub feeder: ::prost::alloc::string::String,
    #[prost(message, repeated, tag="3")]
    pub prices: ::prost::alloc::vec::Vec<ExchangeRateTuple>,
}
/// Emitted by MsgAggregateExchangePrevote when an aggregate prevote is added
/// to state
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventAggregatePrevote {
    /// Validator is the Bech32 address to which the vote will be credited.
    #[prost(string, tag="1")]
    pub validator: ::prost::alloc::string::String,
    /// Feeder is the delegate or representative that will send vote and prevote
    /// transaction messages on behalf of the voting validator.
    #[prost(string, tag="2")]
    pub feeder: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventValidatorPerformance {
    /// Validator is the Bech32 address to which the vote will be credited.
    #[prost(string, tag="1")]
    pub validator: ::prost::alloc::string::String,
    /// Tendermint consensus voting power
    #[prost(int64, tag="2")]
    pub voting_power: i64,
    /// RewardWeight: Weight of rewards the validator should receive in units of
    /// consensus power.
    #[prost(int64, tag="3")]
    pub reward_weight: i64,
    /// Number of valid votes for which the validator will be rewarded
    #[prost(int64, tag="4")]
    pub win_count: i64,
    /// Number of abstained votes for which there will be no reward or punishment
    #[prost(int64, tag="5")]
    pub abstain_count: i64,
    /// Number of invalid/punishable votes
    #[prost(int64, tag="6")]
    pub miss_count: i64,
}
/// GenesisState defines the oracle module's genesis state.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
    #[prost(message, repeated, tag="2")]
    pub feeder_delegations: ::prost::alloc::vec::Vec<FeederDelegation>,
    #[prost(message, repeated, tag="3")]
    pub exchange_rates: ::prost::alloc::vec::Vec<ExchangeRateTuple>,
    #[prost(message, repeated, tag="4")]
    pub miss_counters: ::prost::alloc::vec::Vec<MissCounter>,
    #[prost(message, repeated, tag="5")]
    pub aggregate_exchange_rate_prevotes: ::prost::alloc::vec::Vec<AggregateExchangeRatePrevote>,
    #[prost(message, repeated, tag="6")]
    pub aggregate_exchange_rate_votes: ::prost::alloc::vec::Vec<AggregateExchangeRateVote>,
    #[prost(string, repeated, tag="7")]
    pub pairs: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    #[prost(message, repeated, tag="8")]
    pub rewards: ::prost::alloc::vec::Vec<Rewards>,
}
/// FeederDelegation is the address for where oracle feeder authority are
/// delegated to. By default this struct is only used at genesis to feed in
/// default feeder addresses.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct FeederDelegation {
    #[prost(string, tag="1")]
    pub feeder_address: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub validator_address: ::prost::alloc::string::String,
}
/// MissCounter defines an miss counter and validator address pair used in
/// oracle module's genesis state
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MissCounter {
    #[prost(string, tag="1")]
    pub validator_address: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub miss_counter: u64,
}
/// QueryExchangeRateRequest is the request type for the Query/ExchangeRate RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExchangeRateRequest {
    /// pair defines the pair to query for.
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
}
/// QueryExchangeRateResponse is response type for the
/// Query/ExchangeRate RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExchangeRateResponse {
    /// exchange_rate defines the exchange rate of assets voted by validators
    #[prost(string, tag="1")]
    pub exchange_rate: ::prost::alloc::string::String,
    /// Block timestamp for the block where the oracle came to consensus for this
    /// price. This timestamp is a conventional Unix millisecond time, i.e. the
    /// number of milliseconds elapsed since January 1, 1970 UTC. 
    #[prost(int64, tag="2")]
    pub block_timestamp_ms: i64,
    /// Block height when the oracle came to consensus for this price.
    #[prost(uint64, tag="3")]
    pub block_height: u64,
}
/// QueryExchangeRatesRequest is the request type for the Query/ExchangeRates RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExchangeRatesRequest {
}
/// QueryExchangeRatesResponse is response type for the
/// Query/ExchangeRates RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExchangeRatesResponse {
    /// exchange_rates defines a list of the exchange rate for all whitelisted
    /// pairs.
    #[prost(message, repeated, tag="1")]
    pub exchange_rates: ::prost::alloc::vec::Vec<ExchangeRateTuple>,
}
/// QueryActivesRequest is the request type for the Query/Actives RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryActivesRequest {
}
/// QueryActivesResponse is response type for the
/// Query/Actives RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryActivesResponse {
    /// actives defines a list of the pair which oracle prices agreed upon.
    #[prost(string, repeated, tag="1")]
    pub actives: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// QueryVoteTargetsRequest is the request type for the Query/VoteTargets RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryVoteTargetsRequest {
}
/// QueryVoteTargetsResponse is response type for the
/// Query/VoteTargets RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryVoteTargetsResponse {
    /// vote_targets defines a list of the pairs in which everyone
    /// should vote in the current vote period.
    #[prost(string, repeated, tag="1")]
    pub vote_targets: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// QueryFeederDelegationRequest is the request type for the
/// Query/FeederDelegation RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeederDelegationRequest {
    /// validator defines the validator address to query for.
    #[prost(string, tag="1")]
    pub validator_addr: ::prost::alloc::string::String,
}
/// QueryFeederDelegationResponse is response type for the
/// Query/FeederDelegation RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeederDelegationResponse {
    /// feeder_addr defines the feeder delegation of a validator
    #[prost(string, tag="1")]
    pub feeder_addr: ::prost::alloc::string::String,
}
/// QueryMissCounterRequest is the request type for the Query/MissCounter RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryMissCounterRequest {
    /// validator defines the validator address to query for.
    #[prost(string, tag="1")]
    pub validator_addr: ::prost::alloc::string::String,
}
/// QueryMissCounterResponse is response type for the
/// Query/MissCounter RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryMissCounterResponse {
    /// miss_counter defines the oracle miss counter of a validator
    #[prost(uint64, tag="1")]
    pub miss_counter: u64,
}
/// QueryAggregatePrevoteRequest is the request type for the
/// Query/AggregatePrevote RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregatePrevoteRequest {
    /// validator defines the validator address to query for.
    #[prost(string, tag="1")]
    pub validator_addr: ::prost::alloc::string::String,
}
/// QueryAggregatePrevoteResponse is response type for the
/// Query/AggregatePrevote RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregatePrevoteResponse {
    /// aggregate_prevote defines oracle aggregate prevote submitted by a validator
    /// in the current vote period
    #[prost(message, optional, tag="1")]
    pub aggregate_prevote: ::core::option::Option<AggregateExchangeRatePrevote>,
}
/// QueryAggregatePrevotesRequest is the request type for the
/// Query/AggregatePrevotes RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregatePrevotesRequest {
}
/// QueryAggregatePrevotesResponse is response type for the
/// Query/AggregatePrevotes RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregatePrevotesResponse {
    /// aggregate_prevotes defines all oracle aggregate prevotes submitted in the
    /// current vote period
    #[prost(message, repeated, tag="1")]
    pub aggregate_prevotes: ::prost::alloc::vec::Vec<AggregateExchangeRatePrevote>,
}
/// QueryAggregateVoteRequest is the request type for the Query/AggregateVote RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregateVoteRequest {
    /// validator defines the validator address to query for.
    #[prost(string, tag="1")]
    pub validator_addr: ::prost::alloc::string::String,
}
/// QueryAggregateVoteResponse is response type for the
/// Query/AggregateVote RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregateVoteResponse {
    /// aggregate_vote defines oracle aggregate vote submitted by a validator in
    /// the current vote period
    #[prost(message, optional, tag="1")]
    pub aggregate_vote: ::core::option::Option<AggregateExchangeRateVote>,
}
/// QueryAggregateVotesRequest is the request type for the Query/AggregateVotes
/// RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregateVotesRequest {
}
/// QueryAggregateVotesResponse is response type for the
/// Query/AggregateVotes RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryAggregateVotesResponse {
    /// aggregate_votes defines all oracle aggregate votes submitted in the current
    /// vote period
    #[prost(message, repeated, tag="1")]
    pub aggregate_votes: ::prost::alloc::vec::Vec<AggregateExchangeRateVote>,
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
/// a snapshot of the prices at a given point in time
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PriceSnapshot {
    #[prost(string, tag="1")]
    pub pair: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub price: ::prost::alloc::string::String,
    /// milliseconds since unix epoch
    #[prost(int64, tag="3")]
    pub timestamp_ms: i64,
}
/// MsgAggregateExchangeRatePrevote represents a message to submit
/// aggregate exchange rate prevote.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAggregateExchangeRatePrevote {
    #[prost(string, tag="1")]
    pub hash: ::prost::alloc::string::String,
    /// Feeder is the Bech32 address of the price feeder. A validator may
    /// specify multiple price feeders by delegating them consent. The validator
    /// address is also a valid feeder by default.
    #[prost(string, tag="2")]
    pub feeder: ::prost::alloc::string::String,
    /// Validator is the Bech32 address to which the prevote will be credited.
    #[prost(string, tag="3")]
    pub validator: ::prost::alloc::string::String,
}
/// MsgAggregateExchangeRatePrevoteResponse defines the
/// Msg/AggregateExchangeRatePrevote response type.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAggregateExchangeRatePrevoteResponse {
}
/// MsgAggregateExchangeRateVote represents a message to submit
/// aggregate exchange rate vote.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAggregateExchangeRateVote {
    #[prost(string, tag="1")]
    pub salt: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub exchange_rates: ::prost::alloc::string::String,
    /// Feeder is the Bech32 address of the price feeder. A validator may
    /// specify multiple price feeders by delegating them consent. The validator
    /// address is also a valid feeder by default.
    #[prost(string, tag="3")]
    pub feeder: ::prost::alloc::string::String,
    /// Validator is the Bech32 address to which the vote will be credited.
    #[prost(string, tag="4")]
    pub validator: ::prost::alloc::string::String,
}
/// MsgAggregateExchangeRateVoteResponse defines the
/// Msg/AggregateExchangeRateVote response type.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgAggregateExchangeRateVoteResponse {
}
/// MsgDelegateFeedConsent represents a message to delegate oracle voting rights
/// to another address.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgDelegateFeedConsent {
    #[prost(string, tag="1")]
    pub operator: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub delegate: ::prost::alloc::string::String,
}
/// MsgDelegateFeedConsentResponse defines the Msg/DelegateFeedConsent response
/// type.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgDelegateFeedConsentResponse {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditOracleParams {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub params: ::core::option::Option<OracleParamsMsg>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditOracleParamsResponse {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct OracleParamsMsg {
    /// VotePeriod defines the number of blocks during which voting takes place.
    #[prost(uint64, tag="1")]
    pub vote_period: u64,
    /// VoteThreshold specifies the minimum proportion of votes that must be
    /// received for a ballot to pass.
    #[prost(string, tag="2")]
    pub vote_threshold: ::prost::alloc::string::String,
    /// RewardBand defines a maxium divergence that a price vote can have from the
    /// weighted median in the ballot. If a vote lies within the valid range
    /// defined by:
    /// 	μ := weightedMedian,
    /// 	validRange := μ ± (μ * rewardBand / 2),
    /// then rewards are added to the validator performance.
    /// Note that if the reward band is smaller than 1 standard
    /// deviation, the band is taken to be 1 standard deviation.a price
    #[prost(string, tag="3")]
    pub reward_band: ::prost::alloc::string::String,
    /// The set of whitelisted markets, or asset pairs, for the module.
    /// Ex. '\["unibi:uusd","ubtc:uusd"\]'
    #[prost(string, repeated, tag="4")]
    pub whitelist: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// SlashFraction returns the proportion of an oracle's stake that gets
    /// slashed in the event of slashing. `SlashFraction` specifies the exact
    /// penalty for failing a voting period.
    #[prost(string, tag="5")]
    pub slash_fraction: ::prost::alloc::string::String,
    /// SlashWindow returns the number of voting periods that specify a
    /// "slash window". After each slash window, all oracles that have missed more
    /// than the penalty threshold are slashed. Missing the penalty threshold is
    /// synonymous with submitting fewer valid votes than `MinValidPerWindow`.
    #[prost(uint64, tag="6")]
    pub slash_window: u64,
    #[prost(string, tag="7")]
    pub min_valid_per_window: ::prost::alloc::string::String,
    /// Amount of time to look back for TWAP calculations
    #[prost(message, optional, tag="8")]
    pub twap_lookback_window: ::core::option::Option<::prost_types::Duration>,
    /// The minimum number of voters (i.e. oracle validators) per pair for it to be
    /// considered a passing ballot. Recommended at least 4.
    #[prost(uint64, tag="9")]
    pub min_voters: u64,
    /// The validator fee ratio that is given to validators every epoch.
    #[prost(string, tag="10")]
    pub validator_fee_ratio: ::prost::alloc::string::String,
    #[prost(uint64, tag="11")]
    pub expiration_blocks: u64,
}
// @@protoc_insertion_point(module)
