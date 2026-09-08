// @generated
/// GenesisState represents the messages to be processed during genesis by the genmsg module.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, repeated, tag="1")]
    pub messages: ::prost::alloc::vec::Vec<::prost_types::Any>,
}
// @@protoc_insertion_point(module)
