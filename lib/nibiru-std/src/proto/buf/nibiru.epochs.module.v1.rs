// @generated
/// Module is the config object for the epochs module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Module {
    /// hooks_order specifies the order of staking hooks and should be a list
    /// of module names which provide a staking hooks instance. If no order is
    /// provided, then hooks will be applied in alphabetical order of module names.
    #[prost(string, repeated, tag="1")]
    pub hooks_order: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// authority defines the custom module authority. If not set, defaults to the governance module.
    #[prost(string, tag="2")]
    pub authority: ::prost::alloc::string::String,
}
/// ModuleAccountPermission represents permissions for a module account.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ModuleAccountPermission {
    /// account is the name of the module.
    #[prost(string, tag="1")]
    pub account: ::prost::alloc::string::String,
    /// permissions are the permissions this module has. Currently recognized
    /// values are minter, burner and staking.
    #[prost(string, repeated, tag="2")]
    pub permissions: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
// @@protoc_insertion_point(module)
