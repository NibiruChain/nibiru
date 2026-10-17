// @generated
/// EthAccount implements the authtypes.AccountI interface and embeds an
/// authtypes.BaseAccount type. It is compatible with the auth AccountKeeper.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EthAccount {
    /// base_account is an authtypes.BaseAccount
    #[prost(message, optional, tag="1")]
    pub base_account: ::core::option::Option<crate::proto::cosmos::auth::v1beta1::BaseAccount>,
    /// code_hash is the hash calculated from the code contents
    #[prost(string, tag="2")]
    pub code_hash: ::prost::alloc::string::String,
}
/// TxResult is the value stored in eth tx indexer
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct TxResult {
    /// height of the blockchain
    #[prost(int64, tag="1")]
    pub height: i64,
    /// tx_index is the index of the block transaction. It is not the index of an
    /// "internal transaction"
    #[prost(uint32, tag="2")]
    pub tx_index: u32,
    /// msg_index in a batch transaction
    #[prost(uint32, tag="3")]
    pub msg_index: u32,
    /// eth_tx_index is the index in the list of valid eth tx in the block. Said
    /// another way, it is the index of the transaction list returned by
    /// eth_getBlock API.
    #[prost(int32, tag="4")]
    pub eth_tx_index: i32,
    /// failed is true if the eth transaction did not succeed
    #[prost(bool, tag="5")]
    pub failed: bool,
    /// gas_used by the transaction. If it exceeds the block gas limit,
    /// it's set to gas limit, which is what's actually deducted by ante handler.
    #[prost(uint64, tag="6")]
    pub gas_used: u64,
    /// cumulative_gas_used specifies the cumulated amount of gas used for all
    /// processed messages within the current batch transaction.
    #[prost(uint64, tag="7")]
    pub cumulative_gas_used: u64,
}
// @@protoc_insertion_point(module)
