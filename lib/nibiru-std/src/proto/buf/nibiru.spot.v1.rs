// @generated
/// Configuration parameters for the pool.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PoolParams {
    #[prost(string, tag="1")]
    pub swap_fee: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub exit_fee: ::prost::alloc::string::String,
    /// Amplification Parameter (A): Larger value of A make the curve better
    /// resemble a straight line in the center (when pool is near balance).  Highly
    /// volatile assets should use a lower value, while assets that are closer
    /// together may be best with a higher value. This is only used if the
    /// pool_type is set to 1 (stableswap)
    #[prost(string, tag="3")]
    pub a: ::prost::alloc::string::String,
    #[prost(enumeration="PoolType", tag="4")]
    pub pool_type: i32,
}
/// Which assets the pool contains.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PoolAsset {
    /// Coins we are talking about,
    /// the denomination must be unique amongst all PoolAssets for this pool.
    #[prost(message, optional, tag="1")]
    pub token: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// Weight that is not normalized. This weight must be less than 2^50
    #[prost(string, tag="2")]
    pub weight: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pool {
    /// The pool id.
    #[prost(uint64, tag="1")]
    pub id: u64,
    /// The pool account address.
    #[prost(string, tag="2")]
    pub address: ::prost::alloc::string::String,
    /// Fees and other pool-specific parameters.
    #[prost(message, optional, tag="3")]
    pub pool_params: ::core::option::Option<PoolParams>,
    /// These are assumed to be sorted by denomiation.
    /// They contain the pool asset and the information about the weight
    #[prost(message, repeated, tag="4")]
    pub pool_assets: ::prost::alloc::vec::Vec<PoolAsset>,
    /// sum of all non-normalized pool weights
    #[prost(string, tag="5")]
    pub total_weight: ::prost::alloc::string::String,
    /// sum of all LP tokens sent out
    #[prost(message, optional, tag="6")]
    pub total_shares: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// - `balancer`: Balancer are pools defined by the equation xy=k, extended by
/// the weighs introduced by Balancer.
/// - `stableswap`: Stableswap pools are defined by a combination of
/// constant-product and constant-sum pool
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum PoolType {
    Balancer = 0,
    Stableswap = 1,
}
impl PoolType {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            PoolType::Balancer => "BALANCER",
            PoolType::Stableswap => "STABLESWAP",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "BALANCER" => Some(Self::Balancer),
            "STABLESWAP" => Some(Self::Stableswap),
            _ => None,
        }
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventPoolCreated {
    /// the address of the user who created the pool
    #[prost(string, tag="1")]
    pub creator: ::prost::alloc::string::String,
    /// the create pool fee
    #[prost(message, repeated, tag="2")]
    pub fees: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the final state of the pool
    #[prost(message, optional, tag="4")]
    pub final_pool: ::core::option::Option<Pool>,
    /// the amount of pool shares that the user received
    #[prost(message, optional, tag="5")]
    pub final_user_pool_shares: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventPoolJoined {
    /// the address of the user who joined the pool
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
    /// the amount of tokens that the user deposited
    #[prost(message, repeated, tag="2")]
    pub tokens_in: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the amount of pool shares that the user received
    #[prost(message, optional, tag="3")]
    pub pool_shares_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the amount of tokens remaining for the user
    #[prost(message, repeated, tag="4")]
    pub rem_coins: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the final state of the pool
    #[prost(message, optional, tag="5")]
    pub final_pool: ::core::option::Option<Pool>,
    /// the final amount of user pool shares
    #[prost(message, optional, tag="6")]
    pub final_user_pool_shares: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventPoolExited {
    /// the address of the user who exited the pool
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
    /// the amount of pool shares that the user exited with
    #[prost(message, optional, tag="2")]
    pub pool_shares_in: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the amount of tokens returned to the user
    #[prost(message, repeated, tag="3")]
    pub tokens_out: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the amount of fees collected by the pool
    #[prost(message, repeated, tag="4")]
    pub fees: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the final state of the pool
    #[prost(message, optional, tag="5")]
    pub final_pool: ::core::option::Option<Pool>,
    /// the final amount of user pool shares
    #[prost(message, optional, tag="6")]
    pub final_user_pool_shares: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventAssetsSwapped {
    /// the address of the user who swapped tokens
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
    /// the amount of tokens that the user deposited
    #[prost(message, optional, tag="2")]
    pub token_in: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the amount of tokens that the user received
    #[prost(message, optional, tag="3")]
    pub token_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the amount of fees collected by the pool
    #[prost(message, optional, tag="4")]
    pub fee: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// the final state of the pool
    #[prost(message, optional, tag="5")]
    pub final_pool: ::core::option::Option<Pool>,
}
/// Params defines the parameters for the module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// The start pool number, i.e. the first pool number that isn't taken yet.
    #[prost(uint64, tag="1")]
    pub starting_pool_number: u64,
    /// The cost of creating a pool, taken from the pool creator's account.
    #[prost(message, repeated, tag="2")]
    pub pool_creation_fee: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    /// The assets that can be used to create liquidity pools
    #[prost(string, repeated, tag="3")]
    pub whitelisted_asset: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// GenesisState defines the spot module's genesis state.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// params defines all the parameters of the module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
    /// pools defines all the pools of the module.
    #[prost(message, repeated, tag="2")]
    pub pools: ::prost::alloc::vec::Vec<Pool>,
}
/// QueryParamsRequest is request type for the Query/Params RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsRequest {
}
/// QueryParamsResponse is response type for the Query/Params RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsResponse {
    /// params holds all the parameters of this module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolNumberRequest {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolNumberResponse {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolResponse {
    #[prost(message, optional, tag="1")]
    pub pool: ::core::option::Option<Pool>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolsRequest {
    /// pagination defines an optional pagination for the request.
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageRequest>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolsResponse {
    #[prost(message, repeated, tag="1")]
    pub pools: ::prost::alloc::vec::Vec<Pool>,
    /// pagination defines the pagination in the response.
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageResponse>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolParamsRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPoolParamsResponse {
    #[prost(message, optional, tag="1")]
    pub pool_params: ::core::option::Option<PoolParams>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryNumPoolsRequest {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryNumPoolsResponse {
    #[prost(uint64, tag="1")]
    pub num_pools: u64,
}
/// --------------------------------------------
/// Query total liquidity the protocol
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTotalLiquidityRequest {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTotalLiquidityResponse {
    #[prost(message, repeated, tag="1")]
    pub liquidity: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// --------------------------------------------
/// Query total liquidity for a pool
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTotalPoolLiquidityRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTotalPoolLiquidityResponse {
    #[prost(message, repeated, tag="1")]
    pub liquidity: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTotalSharesRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTotalSharesResponse {
    /// sum of all LP tokens sent out
    #[prost(message, optional, tag="1")]
    pub total_shares: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// Returns the amount of tokenInDenom to produce 1 tokenOutDenom
/// For example, if the price of NIBI = 9.123 NUSD, then setting
/// tokenInDenom=NUSD and tokenOutDenom=NIBI would give "9.123".
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySpotPriceRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
    /// the denomination of the token you are giving into the pool
    #[prost(string, tag="2")]
    pub token_in_denom: ::prost::alloc::string::String,
    /// the denomination of the token you are taking out of the pool
    #[prost(string, tag="3")]
    pub token_out_denom: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySpotPriceResponse {
    #[prost(string, tag="1")]
    pub spot_price: ::prost::alloc::string::String,
}
/// Given an exact amount of tokens in and a target tokenOutDenom, calculates
/// the expected amount of tokens out received from a swap.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySwapExactAmountInRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
    #[prost(message, optional, tag="2")]
    pub token_in: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(string, tag="3")]
    pub token_out_denom: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySwapExactAmountInResponse {
    #[prost(message, optional, tag="2")]
    pub token_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(message, optional, tag="3")]
    pub fee: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
/// Given an exact amount of tokens out and a target tokenInDenom, calculates
/// the expected amount of tokens in required to do the swap.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySwapExactAmountOutRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
    #[prost(message, optional, tag="2")]
    pub token_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(string, tag="3")]
    pub token_in_denom: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySwapExactAmountOutResponse {
    #[prost(message, optional, tag="2")]
    pub token_in: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryJoinExactAmountInRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
    #[prost(message, repeated, tag="2")]
    pub tokens_in: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryJoinExactAmountInResponse {
    /// amount of pool shares returned to user after join
    #[prost(string, tag="1")]
    pub pool_shares_out: ::prost::alloc::string::String,
    /// coins remaining after pool join
    #[prost(message, repeated, tag="2")]
    pub rem_coins: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryJoinExactAmountOutRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryJoinExactAmountOutResponse {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExitExactAmountInRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
    /// amount of pool shares to return to pool
    #[prost(string, tag="2")]
    pub pool_shares_in: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExitExactAmountInResponse {
    /// coins obtained after exiting
    #[prost(message, repeated, tag="1")]
    pub tokens_out: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(message, repeated, tag="2")]
    pub fees: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExitExactAmountOutRequest {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryExitExactAmountOutResponse {
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCreatePool {
    #[prost(string, tag="1")]
    pub creator: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub pool_params: ::core::option::Option<PoolParams>,
    #[prost(message, repeated, tag="3")]
    pub pool_assets: ::prost::alloc::vec::Vec<PoolAsset>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCreatePoolResponse {
    #[prost(uint64, tag="1")]
    pub pool_id: u64,
}
///
/// Message to join a pool (identified by poolId) with a set of tokens to deposit.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgJoinPool {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub pool_id: u64,
    #[prost(message, repeated, tag="3")]
    pub tokens_in: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(bool, tag="4")]
    pub use_all_coins: bool,
}
///
/// Response when a user joins a pool.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgJoinPoolResponse {
    /// the final state of the pool after a join
    #[prost(message, optional, tag="1")]
    pub pool: ::core::option::Option<Pool>,
    /// sum of LP tokens minted from the join
    #[prost(message, optional, tag="2")]
    pub num_pool_shares_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// remaining tokens from attempting to join the pool
    #[prost(message, repeated, tag="3")]
    pub remaining_coins: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgExitPool {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub pool_id: u64,
    #[prost(message, optional, tag="3")]
    pub pool_shares: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgExitPoolResponse {
    #[prost(message, repeated, tag="3")]
    pub tokens_out: ::prost::alloc::vec::Vec<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSwapAssets {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(uint64, tag="2")]
    pub pool_id: u64,
    #[prost(message, optional, tag="3")]
    pub token_in: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(string, tag="4")]
    pub token_out_denom: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSwapAssetsResponse {
    #[prost(message, optional, tag="3")]
    pub token_out: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
// @@protoc_insertion_point(module)
