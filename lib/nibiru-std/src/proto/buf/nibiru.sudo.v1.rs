// @generated
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Sudoers {
    /// Root: The "root" user.
    #[prost(string, tag="1")]
    pub root: ::prost::alloc::string::String,
    /// Contracts: The set of contracts with elevated permissions.
    #[prost(string, repeated, tag="2")]
    pub contracts: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// GenesisState: State for migrations and genesis for the x/sudo module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, optional, tag="1")]
    pub sudoers: ::core::option::Option<Sudoers>,
}
/// EventUpdateSudoers: ABCI event emitted upon execution of "MsgEditSudoers".
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventUpdateSudoers {
    #[prost(message, optional, tag="1")]
    pub sudoers: ::core::option::Option<Sudoers>,
    /// Action is the type of update that occurred to the "sudoers"
    #[prost(string, tag="2")]
    pub action: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySudoersRequest {
}
/// QuerySudoersResponse indicates the successful execution of MsgEditSudeors.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySudoersResponse {
    #[prost(message, optional, tag="1")]
    pub sudoers: ::core::option::Option<Sudoers>,
}
// -------------------------- EditSudoers --------------------------

/// MsgEditSudoers: Msg to update the "Sudoers" state. 
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditSudoers {
    /// Action: identifier for the type of edit that will take place. Using this
    ///    action field prevents us from needing to create several similar message
    ///    types.
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
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgEditSudoersResponse {
}
// -------------------------- ChangeRoot --------------------------

/// MsgChangeRoot: Msg to update the "Sudoers" state. 
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
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeRootResponse {
}
// @@protoc_insertion_point(module)
