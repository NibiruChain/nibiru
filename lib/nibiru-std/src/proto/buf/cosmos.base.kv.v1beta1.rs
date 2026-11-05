// @generated
/// Pairs defines a repeated slice of Pair objects.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pairs {
    #[prost(message, repeated, tag="1")]
    pub pairs: ::prost::alloc::vec::Vec<Pair>,
}
/// Pair defines a key/value bytes tuple.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pair {
    #[prost(bytes="bytes", tag="1")]
    pub key: ::prost::bytes::Bytes,
    #[prost(bytes="bytes", tag="2")]
    pub value: ::prost::bytes::Bytes,
}
// @@protoc_insertion_point(module)
