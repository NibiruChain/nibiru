// @generated
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventCreateDenom {
    #[prost(string, tag="1")]
    pub denom: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub creator: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventChangeAdmin {
    #[prost(string, tag="1")]
    pub denom: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub new_admin: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub old_admin: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventMint {
    #[prost(message, optional, tag="1")]
    pub coin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(string, tag="2")]
    pub to_addr: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub caller: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventBurn {
    #[prost(message, optional, tag="1")]
    pub coin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    #[prost(string, tag="2")]
    pub from_addr: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub caller: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EventSetDenomMetadata {
    #[prost(string, tag="1")]
    pub denom: ::prost::alloc::string::String,
    /// Metadata: Official x/bank metadata for the denom. All token factory denoms
    /// are standard, native assets. The "metadata.base" is the denom.
    #[prost(message, optional, tag="2")]
    pub metadata: ::core::option::Option<crate::proto::cosmos::bank::v1beta1::Metadata>,
    #[prost(string, tag="3")]
    pub caller: ::prost::alloc::string::String,
}
/// DenomAuthorityMetadata specifies metadata foraddresses that have specific
/// capabilities over a token factory denom. Right now there is only one Admin
/// permission, but is planned to be extended to the future.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct DenomAuthorityMetadata {
    /// Admin: Bech32 address of the admin for the tokefactory denom. Can be empty
    /// for no admin.
    #[prost(string, tag="1")]
    pub admin: ::prost::alloc::string::String,
}
/// ModuleParams defines the parameters for the tokenfactory module.
///
/// ### On Denom Creation Costs
///
/// We'd like for fees to be paid by the user/signer of a ransaction, but in many
/// casess, token creation is abstracted away behind a smart contract. Setting a
/// nonzero `denom_creation_fee` would force each contract to handle collecting
/// and paying a fees for denom (factory/{contract-addr}/{subdenom}) creation on
/// behalf of the end user.
///
/// For IBC token transfers, it's unclear who should pay the fee—the contract,
/// the relayer, or the original sender?
/// > "Charging fees will mess up composability, the same way Terra transfer tax
/// > caused all kinds of headaches for contract devs." - @ethanfrey
///
/// ### Recommended Solution
///
/// Have the end user (signer) pay fees directly in the form of higher gas costs.
/// This way, contracts won't need to handle collecting or paying fees. And for
/// IBC, the gas costs are already paid by the original sender and can be
/// estimated by the relayer. It's easier to tune gas costs to make spam
/// prohibitively expensive since there are per-transaction and per-block gas
/// limits.
///
/// See <https://github.com/CosmWasm/token-factory/issues/11> for the initial
/// discussion of the issue with @ethanfrey and @valardragon.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ModuleParams {
    /// Adds gas consumption to the execution of `MsgCreateDenom` as a method of
    /// spam prevention. Defaults to 10 NIBI.
    #[prost(uint64, tag="1")]
    pub denom_creation_gas_consume: u64,
}
/// TFDenom is a token factory (TF) denom. The canonical representation is
/// "tf/{creator}/{subdenom}", its unique denomination in the x/bank module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct TfDenom {
    /// Creator: Bech32 address of the creator of the denom.
    #[prost(string, tag="1")]
    pub creator: ::prost::alloc::string::String,
    /// Subdenom: Unique suffix of a token factory denom. A subdenom is specific
    /// to a given creator. It is the name given during a token factory "Mint".
    #[prost(string, tag="2")]
    pub subdenom: ::prost::alloc::string::String,
}
// ----------------------------------------------
// Genesis
// ----------------------------------------------

/// GenesisState for the Token Factory module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<ModuleParams>,
    #[prost(message, repeated, tag="2")]
    pub factory_denoms: ::prost::alloc::vec::Vec<GenesisDenom>,
}
/// GenesisDenom defines a tokenfactory denoms in the genesis state.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisDenom {
    #[prost(string, tag="1")]
    pub denom: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub authority_metadata: ::core::option::Option<DenomAuthorityMetadata>,
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
    /// Module parameters stored in state
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<ModuleParams>,
}
/// QueryDenomsRequest: gRPC query for all denoms registered for a creator
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryDenomsRequest {
    #[prost(string, tag="1")]
    pub creator: ::prost::alloc::string::String,
}
/// QueryDenomsResponse: All registered denoms for a creator
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryDenomsResponse {
    #[prost(string, repeated, tag="1")]
    pub denoms: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
/// QueryDenomInfoRequest: gRPC query for the denom admin and x/bank metadata
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryDenomInfoRequest {
    #[prost(string, tag="1")]
    pub denom: ::prost::alloc::string::String,
}
/// QueryDenomInfoResponse: All registered denoms for a creator
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryDenomInfoResponse {
    /// Admin of the token factory denom
    #[prost(string, tag="1")]
    pub admin: ::prost::alloc::string::String,
    /// Metadata: Official x/bank metadata for the denom. All token factory denoms
    /// are standard, native assets.
    #[prost(message, optional, tag="2")]
    pub metadata: ::core::option::Option<crate::proto::cosmos::bank::v1beta1::Metadata>,
}
/// MsgCreateDenom: sdk.Msg that registers an a token factory denom.
/// A denom has the form "tf/\[creatorAddr]/[subdenom\]".
///   - Denoms become unique x/bank tokens, so the creator-subdenom pair that
///     defines a denom cannot be reused.
///   - The resulting denom's admin is originally set to be the creator, but the
///     admin can be changed later.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCreateDenom {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// subdenom can be up to 44 "alphanumeric" characters long.
    #[prost(string, tag="2")]
    pub subdenom: ::prost::alloc::string::String,
}
/// MsgCreateDenomResponse is the return value of MsgCreateDenom
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCreateDenomResponse {
    /// NewTokenDenom: identifier for the newly created token factory denom.
    #[prost(string, tag="1")]
    pub new_token_denom: ::prost::alloc::string::String,
}
/// MsgChangeAdmin is the sdk.Msg type for allowing an admin account to change
/// admin of a denom to a new account
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeAdmin {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub denom: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub new_admin: ::prost::alloc::string::String,
}
/// MsgChangeAdminResponse is the gRPC response for the MsgChangeAdmin TxMsg.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgChangeAdminResponse {
}
/// MsgUpdateModuleParams: sdk.Msg for updating the x/tokenfactory module params
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateModuleParams {
    /// Authority: Address of the governance module account.
    #[prost(string, tag="1")]
    pub authority: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub params: ::core::option::Option<ModuleParams>,
}
/// MsgUpdateModuleParamsResponse is the gRPC response for the
/// MsgUpdateModuleParams TxMsg.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateModuleParamsResponse {
}
/// MsgMint: sdk.Msg (TxMsg) where an denom admin mints more of the token.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgMint {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// coin: The denom identifier and amount to mint.
    #[prost(message, optional, tag="2")]
    pub coin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// mint_to_addr: An address to which tokens will be minted. If blank,
    /// tokens are minted to the "sender".
    #[prost(string, tag="3")]
    pub mint_to: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgMintResponse {
    #[prost(string, tag="1")]
    pub mint_to: ::prost::alloc::string::String,
}
/// MsgBurn: sdk.Msg (TxMsg) where a denom admin burns some of the token.
/// The reason that the sender isn't automatically the "burn_from" address
/// is to support smart contracts (primary use case). In this situation, the
/// contract is the message signer and sender, while "burn_from" is based on the
/// contract logic.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgBurn {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// coin: The denom identifier and amount to burn.
    #[prost(message, optional, tag="2")]
    pub coin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
    /// burn_from: The address from which tokens will be burned.
    #[prost(string, tag="3")]
    pub burn_from: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgBurnResponse {
}
/// MsgSetDenomMetadata: sdk.Msg (TxMsg) enabling the denom admin to change its
/// bank metadata.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSetDenomMetadata {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// Metadata: Official x/bank metadata for the denom. All token factory denoms
    /// are standard, native assets. The "metadata.base" is the denom.
    #[prost(message, optional, tag="2")]
    pub metadata: ::core::option::Option<crate::proto::cosmos::bank::v1beta1::Metadata>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSetDenomMetadataResponse {
}
/// MsgSudoSetDenomMetadata: sdk.Msg (TxMsg) enabling Nibiru's "sudoers" to change
/// bank metadata.
/// \[SUDO\] Only callable by sudoers.
///
/// Use Cases:
///    - To define metadata for ICS20 assets brought
///      over to the chain via IBC, as they don't have metadata by default.
///    - To set metadata for Bank Coins created via the Token Factory
///      module in case the admin forgets to do so. This is important because of
///      the relationship Token Factory assets can have with ERC20s with the
///      [FunToken Mechanism].
///
/// [FunToken Mechanism]: <https://nibiru.fi/docs/evm/funtoken.html>
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSudoSetDenomMetadata {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    /// Metadata: Official x/bank metadata for the denom. The "metadata.base" is
    /// the denom.
    #[prost(message, optional, tag="2")]
    pub metadata: ::core::option::Option<crate::proto::cosmos::bank::v1beta1::Metadata>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgSudoSetDenomMetadataResponse {
}
/// Burn a native token such as unibi
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgBurnNative {
    #[prost(string, tag="1")]
    pub sender: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub coin: ::core::option::Option<crate::proto::cosmos::base::v1beta1::Coin>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgBurnNativeResponse {
}
// @@protoc_insertion_point(module)
