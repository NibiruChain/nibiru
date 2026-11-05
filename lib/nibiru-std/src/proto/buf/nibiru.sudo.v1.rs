// @generated
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Sudoers {
    /// Root: The "root" user.
    #[prost(string, tag="1")]
    pub root: ::prost::alloc::string::String,
    /// Roles grant specific permissions to account or contract addresses.
    #[prost(message, repeated, tag="3")]
    pub roles: ::prost::alloc::vec::Vec<RoleMembers>,
}
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct RoleMembers {
    #[prost(string, tag="1")]
    pub role: ::prost::alloc::string::String,
    #[prost(string, repeated, tag="2")]
    pub members: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// GenesisState: State for migrations and genesis for the x/sudo module.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, optional, tag="1")]
    pub sudoers: ::core::option::Option<Sudoers>,
    #[prost(message, optional, tag="2")]
    pub zero_gas_actors: ::core::option::Option<ZeroGasActors>,
    #[prost(string, tag="3")]
    pub wasm_block_hooks_contract: ::prost::alloc::string::String,
}
/// ZeroGasActors: Actors that can execute zero gas transactions against a set of
/// smart contracts.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ZeroGasActors {
    /// Senders: Addresses that can sign for zero gas transactions from the
    /// contract set.
    #[prost(string, repeated, tag="1")]
    pub senders: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// Contracts: contract addresses that can be invoked by "senders" with zero
    /// gas costs.
    #[prost(string, repeated, tag="2")]
    pub contracts: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// AlwaysZeroGasContracts: contract addresses that can be invoked with zero
    /// gas costs for any sender.
    #[prost(string, repeated, tag="3")]
    pub always_zero_gas_contracts: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// EventUpdateSudoers: ABCI event emitted upon execution of "MsgEditSudoers".
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventUpdateSudoers {
    #[prost(message, optional, tag="1")]
    pub sudoers: ::core::option::Option<Sudoers>,
    /// Action is the type of update that occurred to the "sudoers"
    #[prost(string, tag="2")]
    pub action: ::prost::alloc::string::String,
}
/// QuerySudoersRequest is the request type for the gRPC query method,
/// "/nibiru.sudo.v1.Query/QuerySudoers".
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySudoersRequest {
}
/// QuerySudoersResponse is the response type for the gRPC query method,
/// "/nibiru.sudo.v1.Query/QuerySudoers"
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySudoersResponse {
    #[prost(message, optional, tag="1")]
    pub sudoers: ::core::option::Option<Sudoers>,
}
/// QueryZeroGasActorsRequest is the request type for the gRPC query method,
/// "/nibiru.sudo.v1.Query/QueryZeroGasActors"
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryZeroGasActorsRequest {
}
/// QueryZeroGasActorsResponse is the response type for the gRPC query method,
/// "/nibiru.sudo.v1.Query/QueryZeroGasActors"
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryZeroGasActorsResponse {
    #[prost(message, optional, tag="1")]
    pub actors: ::core::option::Option<ZeroGasActors>,
}
// -------------------------- EditSudoers --------------------------

/// MsgEditSudoers: Msg to update the "Sudoers" state.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditSudoers {
    /// Action: identifier for the type of edit that will take place. Using this
    /// action field prevents us from needing to create several similar message
    /// types.
    #[prost(string, tag="1")]
    pub action: ::prost::alloc::string::String,
    /// Contracts: An input payload.
    #[prost(string, repeated, tag="2")]
    pub contracts: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// Sender: Address for the signer of the transaction.
    #[prost(string, tag="3")]
    pub sender: ::prost::alloc::string::String,
}
/// MsgEditSudoersResponse indicates the successful execution of MsgEditSudeors.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditSudoersResponse {
}
// -------------------------- ChangeRoot --------------------------

/// MsgChangeRoot: Msg to update the "Sudoers" state.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeRoot {
    /// Sender: Address for the signer of the transaction.
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// NewRoot: New root address.
    #[prost(string, tag="2")]
    pub new_root: ::prost::alloc::string::String,
}
/// MsgChangeRootResponse indicates the successful execution of MsgChangeRoot.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeRootResponse {
}
// -------------------------- EditZeroGasActors --------------------------

/// MsgEditZeroGasActors: Tx msg to update the "ZeroGasActors" state.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditZeroGasActors {
    /// Actors that can execute zero gas transactions against a set of
    /// smart contracts.
    #[prost(message, optional, tag="1")]
    pub actors: ::core::option::Option<ZeroGasActors>,
    /// Sender: Nibiru Bech32 Address for the signer of the transaction.
    #[prost(string, tag="2")]
    pub sender: ::prost::alloc::string::String,
}
/// MsgEditZeroGasActorsResponse indicates the successful execution of
/// MsgEditZeroGasActors.
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditZeroGasActorsResponse {
}
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateRoleMembers {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub role: ::prost::alloc::string::String,
    #[prost(string, repeated, tag="3")]
    pub add: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    #[prost(string, repeated, tag="4")]
    pub remove: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
#[derive(::serde::Serialize)]
#[derive(::serde::Deserialize)]
#[serde(default)]
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateRoleMembersResponse {
}
// @@protoc_insertion_point(module)
