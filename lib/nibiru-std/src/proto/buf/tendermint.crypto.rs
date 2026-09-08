// @generated
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Proof {
    #[prost(int64, tag="1")]
    pub total: i64,
    #[prost(int64, tag="2")]
    pub index: i64,
    #[prost(bytes="bytes", tag="3")]
    pub leaf_hash: ::prost::bytes::Bytes,
    #[prost(bytes="bytes", repeated, tag="4")]
    pub aunts: ::prost::alloc::vec::Vec<::prost::bytes::Bytes>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ValueOp {
    /// Encoded in ProofOp.Key.
    #[prost(bytes="bytes", tag="1")]
    pub key: ::prost::bytes::Bytes,
    /// To encode in ProofOp.Data
    #[prost(message, optional, tag="2")]
    pub proof: ::core::option::Option<Proof>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct DominoOp {
    #[prost(string, tag="1")]
    pub key: ::prost::alloc::string::String,
    #[prost(string, tag="2")]
    pub input: ::prost::alloc::string::String,
    #[prost(string, tag="3")]
    pub output: ::prost::alloc::string::String,
}
/// ProofOp defines an operation used for calculating Merkle root
/// The data could be arbitrary format, providing nessecary data
/// for example neighbouring node hash
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
/// ProofOps is Merkle proof defined by the list of ProofOps
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ProofOps {
    #[prost(message, repeated, tag="1")]
    pub ops: ::prost::alloc::vec::Vec<ProofOp>,
}
/// PublicKey defines the keys available for use with Validators
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PublicKey {
    #[prost(oneof="public_key::Sum", tags="1, 2")]
    pub sum: ::core::option::Option<public_key::Sum>,
}
/// Nested message and enum types in `PublicKey`.
pub mod public_key {
    #[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Oneof)]
    pub enum Sum {
        #[prost(bytes, tag="1")]
        Ed25519(::prost::bytes::Bytes),
        #[prost(bytes, tag="2")]
        Secp256k1(::prost::bytes::Bytes),
    }
}
// @@protoc_insertion_point(module)
