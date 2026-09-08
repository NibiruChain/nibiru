// @generated
/// FeeShare defines an instance that organizes fee distribution conditions for
/// the owner of a given smart contract
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct FeeShare {
    /// contract_address is the bech32 address of a registered contract in string
    /// form
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// deployer_address is the bech32 address of message sender. It must be the
    /// same as the contracts admin address.
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
    /// withdrawer_address is the bech32 address of account receiving the
    /// transaction fees.
    #[prost(string, tag="3")]
    pub withdrawer_address: ::prost::alloc::string::String,
}
/// ABCI event emitted when a deployer registers a contract to receive fee
/// sharing payouts, specifying the deployer, contract, and withdrawer addresses.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventRegisterDevGas {
    /// deployer is the addess of the account that registered the smart contract to
    /// receive dev gas royalties.
    #[prost(string, tag="1")]
    pub deployer: ::prost::alloc::string::String,
    /// Address of the smart contract. This identifies the specific contract
    /// that will receive fee sharing payouts.
    #[prost(string, tag="2")]
    pub contract: ::prost::alloc::string::String,
    /// The address that will receive the fee sharing payouts for the registered
    /// contract. This could be the deployer address or a separate withdrawer
    /// address specified.
    #[prost(string, tag="3")]
    pub withdrawer: ::prost::alloc::string::String,
}
/// ABCI event emitted when a deployer cancels fee sharing for a contract,
/// specifying the deployer and contract addresses.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventCancelDevGas {
    /// deployer is the addess of the account that registered the smart contract to
    /// receive dev gas royalties.
    #[prost(string, tag="1")]
    pub deployer: ::prost::alloc::string::String,
    /// Address of the smart contract. This identifies the specific contract
    /// that will receive fee sharing payouts.
    #[prost(string, tag="2")]
    pub contract: ::prost::alloc::string::String,
}
/// ABCI event emitted when a deployer updates the fee sharing registration for a
/// contract, specifying updated deployer, contract, and/or withdrawer addresses.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventUpdateDevGas {
    /// deployer is the addess of the account that registered the smart contract to
    /// receive dev gas royalties.
    #[prost(string, tag="1")]
    pub deployer: ::prost::alloc::string::String,
    /// Address of the smart contract. This identifies the specific contract
    /// that will receive fee sharing payouts.
    #[prost(string, tag="2")]
    pub contract: ::prost::alloc::string::String,
    /// The address that will receive the fee sharing payouts for the registered
    /// contract. This could be the deployer address or a separate withdrawer
    /// address specified.
    #[prost(string, tag="3")]
    pub withdrawer: ::prost::alloc::string::String,
}
/// ABCI event emitted when fee sharing payouts are made, containing details on
/// the payouts in JSON format.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventPayoutDevGas {
    #[prost(string, tag="1")]
    pub payouts: ::prost::alloc::string::String,
}
/// GenesisState defines the module's genesis state.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// params are the feeshare module parameters
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<ModuleParams>,
    /// FeeShare is a slice of active registered contracts for fee distribution
    #[prost(message, repeated, tag="2")]
    pub fee_share: ::prost::alloc::vec::Vec<FeeShare>,
}
/// ModuleParams defines the params for the devgas module
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ModuleParams {
    /// enable_feeshare defines a parameter to enable the feeshare module
    #[prost(bool, tag="1")]
    pub enable_fee_share: bool,
    /// developer_shares defines the proportion of the transaction fees to be
    /// distributed to the registered contract owner
    #[prost(string, tag="2")]
    pub developer_shares: ::prost::alloc::string::String,
    /// allowed_denoms defines the list of denoms that are allowed to be paid to
    /// the contract withdraw addresses. If said denom is not in the list, the fees
    /// will ONLY be sent to the community pool.
    /// If this list is empty, all denoms are allowed.
    #[prost(string, repeated, tag="3")]
    pub allowed_denoms: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// QueryFeeSharesRequest is the request type for the Query/FeeShares RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeSharesRequest {
    /// TODO feat(devgas): re-implement the paginated version
    /// TODO feat(colletions): add automatic pagination generation
    ///
    /// pagination defines an optional pagination for the request.
    /// cosmos.base.query.v1beta1.PageRequest pagination = 1;
    #[prost(string, tag="1")]
    pub deployer: ::prost::alloc::string::String,
}
// TODO feat(devgas): re-implement the paginated version
// TODO feat(collections): add automatic pagination generation
// Notes for above feat:
// pagination defines an optional pagination for the request.
// cosmos.base.query.v1beta1.PageRequest pagination = 1;
// pagination defines the pagination in the response.
// cosmos.base.query.v1beta1.PageResponse pagination = 2;

/// QueryFeeSharesResponse is the response type for the Query/FeeShares RPC
/// method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeSharesResponse {
    /// FeeShare is the slice of all stored Reveneue for the deployer
    #[prost(message, repeated, tag="1")]
    pub feeshare: ::prost::alloc::vec::Vec<FeeShare>,
}
/// QueryFeeShareRequest is the request type for the Query/FeeShare RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeShareRequest {
    /// contract_address of a registered contract in bech32 format
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
}
/// QueryFeeShareResponse is the response type for the Query/FeeShare RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeShareResponse {
    /// FeeShare is a stored Reveneue for the queried contract
    #[prost(message, optional, tag="1")]
    pub feeshare: ::core::option::Option<FeeShare>,
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
    /// params is the returned FeeShare parameter
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<ModuleParams>,
}
/// QueryFeeSharesByWithdrawerRequest is the request type for the
/// Query/FeeSharesByWithdrawer RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeSharesByWithdrawerRequest {
    /// withdrawer_address in bech32 format
    #[prost(string, tag="1")]
    pub withdrawer_address: ::prost::alloc::string::String,
}
/// QueryFeeSharesByWithdrawerResponse is the response type for the
/// Query/FeeSharesByWithdrawer RPC method.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeSharesByWithdrawerResponse {
    #[prost(message, repeated, tag="1")]
    pub feeshare: ::prost::alloc::vec::Vec<FeeShare>,
}
/// MsgRegisterFeeShare defines a message that registers a FeeShare
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgRegisterFeeShare {
    /// contract_address in bech32 format
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// deployer_address is the bech32 address of message sender. It must be the
    /// same the contract's admin address
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
    /// withdrawer_address is the bech32 address of account receiving the
    /// transaction fees
    #[prost(string, tag="3")]
    pub withdrawer_address: ::prost::alloc::string::String,
}
/// MsgRegisterFeeShareResponse defines the MsgRegisterFeeShare response type
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgRegisterFeeShareResponse {
}
/// MsgUpdateFeeShare defines a message that updates the withdrawer address for a
/// registered FeeShare
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateFeeShare {
    /// contract_address in bech32 format
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// deployer_address is the bech32 address of message sender. It must be the
    /// same the contract's admin address
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
    /// withdrawer_address is the bech32 address of account receiving the
    /// transaction fees
    #[prost(string, tag="3")]
    pub withdrawer_address: ::prost::alloc::string::String,
}
/// MsgUpdateFeeShareResponse defines the MsgUpdateFeeShare response type
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateFeeShareResponse {
}
/// MsgCancelFeeShare defines a message that cancels a registered FeeShare
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCancelFeeShare {
    /// contract_address in bech32 format
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// deployer_address is the bech32 address of message sender. It must be the
    /// same the contract's admin address
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
}
/// MsgCancelFeeShareResponse defines the MsgCancelFeeShare response type
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCancelFeeShareResponse {
}
/// MsgUpdateParams is the Msg/UpdateParams request type.
///
/// Since: cosmos-sdk 0.47
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateParams {
    /// authority is the address that controls the module (defaults to x/gov unless
    /// overwritten).
    #[prost(string, tag="1")]
    pub authority: ::prost::alloc::string::String,
    /// params defines the x/feeshare parameters to update.
    ///
    /// NOTE: All parameters must be supplied.
    #[prost(message, optional, tag="2")]
    pub params: ::core::option::Option<ModuleParams>,
}
/// MsgUpdateParamsResponse defines the response structure for executing a
/// MsgUpdateParams message.
///
/// Since: cosmos-sdk 0.47
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateParamsResponse {
}
// @@protoc_insertion_point(module)
