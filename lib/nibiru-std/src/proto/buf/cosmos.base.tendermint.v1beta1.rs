// @generated
/// Block is tendermint type Block, with the Header proposer address
/// field converted to bech32 string.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Block {
    #[prost(message, optional, tag="1")]
    pub header: ::core::option::Option<Header>,
    #[prost(message, optional, tag="2")]
    pub data: ::core::option::Option<crate::proto::tendermint::types::Data>,
    #[prost(message, optional, tag="3")]
    pub evidence: ::core::option::Option<crate::proto::tendermint::types::EvidenceList>,
    #[prost(message, optional, tag="4")]
    pub last_commit: ::core::option::Option<crate::proto::tendermint::types::Commit>,
}
/// Header defines the structure of a Tendermint block header.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Header {
    /// basic block info
    #[prost(message, optional, tag="1")]
    pub version: ::core::option::Option<crate::proto::tendermint::version::Consensus>,
    #[prost(string, tag="2")]
    pub chain_id: ::prost::alloc::string::String,
    #[prost(int64, tag="3")]
    pub height: i64,
    #[prost(message, optional, tag="4")]
    pub time: ::core::option::Option<::prost_types::Timestamp>,
    /// prev block info
    #[prost(message, optional, tag="5")]
    pub last_block_id: ::core::option::Option<crate::proto::tendermint::types::BlockId>,
    /// hashes of block data
    ///
    /// commit from validators from the last block
    #[prost(bytes="bytes", tag="6")]
    pub last_commit_hash: ::prost::bytes::Bytes,
    /// transactions
    #[prost(bytes="bytes", tag="7")]
    pub data_hash: ::prost::bytes::Bytes,
    /// hashes from the app output from the prev block
    ///
    /// validators for the current block
    #[prost(bytes="bytes", tag="8")]
    pub validators_hash: ::prost::bytes::Bytes,
    /// validators for the next block
    #[prost(bytes="bytes", tag="9")]
    pub next_validators_hash: ::prost::bytes::Bytes,
    /// consensus params for current block
    #[prost(bytes="bytes", tag="10")]
    pub consensus_hash: ::prost::bytes::Bytes,
    /// state after txs from the previous block
    #[prost(bytes="bytes", tag="11")]
    pub app_hash: ::prost::bytes::Bytes,
    /// root hash of all results from the txs from the previous block
    #[prost(bytes="bytes", tag="12")]
    pub last_results_hash: ::prost::bytes::Bytes,
    /// consensus info
    ///
    /// evidence included in the block
    #[prost(bytes="bytes", tag="13")]
    pub evidence_hash: ::prost::bytes::Bytes,
    /// proposer_address is the original block proposer address, formatted as a Bech32 string.
    /// In Tendermint, this type is `bytes`, but in the SDK, we convert it to a Bech32 string
    /// for better UX.
    ///
    /// original proposer of the block
    #[prost(string, tag="14")]
    pub proposer_address: ::prost::alloc::string::String,
}
/// GetValidatorSetByHeightRequest is the request type for the Query/GetValidatorSetByHeight RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetValidatorSetByHeightRequest {
    #[prost(int64, tag="1")]
    pub height: i64,
    /// pagination defines an pagination for the request.
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// GetValidatorSetByHeightResponse is the response type for the Query/GetValidatorSetByHeight RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetValidatorSetByHeightResponse {
    #[prost(int64, tag="1")]
    pub block_height: i64,
    #[prost(message, repeated, tag="2")]
    pub validators: ::prost::alloc::vec::Vec<Validator>,
    /// pagination defines an pagination for the response.
    #[prost(message, optional, tag="3")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageResponse>,
}
/// GetLatestValidatorSetRequest is the request type for the Query/GetValidatorSetByHeight RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLatestValidatorSetRequest {
    /// pagination defines an pagination for the request.
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// GetLatestValidatorSetResponse is the response type for the Query/GetValidatorSetByHeight RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLatestValidatorSetResponse {
    #[prost(int64, tag="1")]
    pub block_height: i64,
    #[prost(message, repeated, tag="2")]
    pub validators: ::prost::alloc::vec::Vec<Validator>,
    /// pagination defines an pagination for the response.
    #[prost(message, optional, tag="3")]
    pub pagination: ::core::option::Option<crate::proto::cosmos::base::query::v1beta1::PageResponse>,
}
/// Validator is the type for the validator-set.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Validator {
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub pub_key: ::core::option::Option<::prost_types::Any>,
    #[prost(int64, tag="3")]
    pub voting_power: i64,
    #[prost(int64, tag="4")]
    pub proposer_priority: i64,
}
/// GetBlockByHeightRequest is the request type for the Query/GetBlockByHeight RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetBlockByHeightRequest {
    #[prost(int64, tag="1")]
    pub height: i64,
}
/// GetBlockByHeightResponse is the response type for the Query/GetBlockByHeight RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetBlockByHeightResponse {
    #[prost(message, optional, tag="1")]
    pub block_id: ::core::option::Option<crate::proto::tendermint::types::BlockId>,
    /// Deprecated: please use `sdk_block` instead
    #[prost(message, optional, tag="2")]
    pub block: ::core::option::Option<crate::proto::tendermint::types::Block>,
    /// Since: cosmos-sdk 0.47
    #[prost(message, optional, tag="3")]
    pub sdk_block: ::core::option::Option<Block>,
}
/// GetLatestBlockRequest is the request type for the Query/GetLatestBlock RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLatestBlockRequest {
}
/// GetLatestBlockResponse is the response type for the Query/GetLatestBlock RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLatestBlockResponse {
    #[prost(message, optional, tag="1")]
    pub block_id: ::core::option::Option<crate::proto::tendermint::types::BlockId>,
    /// Deprecated: please use `sdk_block` instead
    #[prost(message, optional, tag="2")]
    pub block: ::core::option::Option<crate::proto::tendermint::types::Block>,
    /// Since: cosmos-sdk 0.47
    #[prost(message, optional, tag="3")]
    pub sdk_block: ::core::option::Option<Block>,
}
/// GetSyncingRequest is the request type for the Query/GetSyncing RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetSyncingRequest {
}
/// GetSyncingResponse is the response type for the Query/GetSyncing RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetSyncingResponse {
    #[prost(bool, tag="1")]
    pub syncing: bool,
}
/// GetNodeInfoRequest is the request type for the Query/GetNodeInfo RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetNodeInfoRequest {
}
/// GetNodeInfoResponse is the response type for the Query/GetNodeInfo RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetNodeInfoResponse {
    #[prost(message, optional, tag="1")]
    pub default_node_info: ::core::option::Option<crate::proto::tendermint::p2p::DefaultNodeInfo>,
    #[prost(message, optional, tag="2")]
    pub application_version: ::core::option::Option<VersionInfo>,
}
/// VersionInfo is the type for the GetNodeInfoResponse message.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct VersionInfo {
    #[prost(string, tag="1")]
    pub name: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub app_name: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub version: ::prost::alloc::string::String,
    #[prost(string, tag="4")]
    pub git_commit: ::prost::alloc::string::String,
    #[prost(string, tag="5")]
    pub build_tags: ::prost::alloc::string::String,
    #[prost(string, tag="6")]
    pub go_version: ::prost::alloc::string::String,
    #[prost(message, repeated, tag="7")]
    pub build_deps: ::prost::alloc::vec::Vec<Module>,
    /// Since: cosmos-sdk 0.43
    #[prost(string, tag="8")]
    pub cosmos_sdk_version: ::prost::alloc::string::String,
}
/// Module is the type for VersionInfo
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Module {
    /// module path
    #[prost(string, tag="1")]
    pub path: ::prost::alloc::string::String,
    /// module version
    #[prost(string, tag="2")]
    pub version: ::prost::alloc::string::String,
    /// checksum
    #[prost(string, tag="3")]
    pub sum: ::prost::alloc::string::String,
}
/// ABCIQueryRequest defines the request structure for the ABCIQuery gRPC query.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AbciQueryRequest {
    #[prost(bytes="bytes", tag="1")]
    pub data: ::prost::bytes::Bytes,
    #[prost(string, tag="2")]
    pub path: ::prost::alloc::string::String,
    #[prost(int64, tag="3")]
    pub height: i64,
    #[prost(bool, tag="4")]
    pub prove: bool,
}
/// ABCIQueryResponse defines the response structure for the ABCIQuery gRPC query.
///
/// Note: This type is a duplicate of the ResponseQuery proto type defined in
/// Tendermint.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AbciQueryResponse {
    #[prost(uint32, tag="1")]
    pub code: u32,
    /// nondeterministic
    #[prost(string, tag="3")]
    pub log: ::prost::alloc::string::String,
    /// nondeterministic
    #[prost(string, tag="4")]
    pub info: ::prost::alloc::string::String,
    #[prost(int64, tag="5")]
    pub index: i64,
    #[prost(bytes="bytes", tag="6")]
    pub key: ::prost::bytes::Bytes,
    #[prost(bytes="bytes", tag="7")]
    pub value: ::prost::bytes::Bytes,
    #[prost(message, optional, tag="8")]
    pub proof_ops: ::core::option::Option<ProofOps>,
    #[prost(int64, tag="9")]
    pub height: i64,
    #[prost(string, tag="10")]
    pub codespace: ::prost::alloc::string::String,
}
/// ProofOp defines an operation used for calculating Merkle root. The data could
/// be arbitrary format, providing necessary data for example neighbouring node
/// hash.
///
/// Note: This type is a duplicate of the ProofOp proto type defined in Tendermint.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ProofOp {
    #[prost(string, tag="1")]
    pub r#type: ::prost::alloc::string::String,
    #[prost(bytes="bytes", tag="2")]
    pub key: ::prost::bytes::Bytes,
    #[prost(bytes="bytes", tag="3")]
    pub data: ::prost::bytes::Bytes,
}
/// ProofOps is Merkle proof defined by the list of ProofOps.
///
/// Note: This type is a duplicate of the ProofOps proto type defined in Tendermint.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ProofOps {
    #[prost(message, repeated, tag="1")]
    pub ops: ::prost::alloc::vec::Vec<ProofOp>,
}
// @@protoc_insertion_point(module)
