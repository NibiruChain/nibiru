// @generated
/// PubKey defines a secp256r1 ECDSA public key.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PubKey {
    /// Point on secp256r1 curve in a compressed representation as specified in section
    /// 4.3.6 of ANSI X9.62: <https://webstore.ansi.org/standards/ascx9/ansix9621998>
    #[prost(bytes="bytes", tag="1")]
    pub key: ::prost::bytes::Bytes,
}
/// PrivKey defines a secp256r1 ECDSA private key.
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PrivKey {
    /// secret number serialized using big-endian encoding
    #[prost(bytes="bytes", tag="1")]
    pub secret: ::prost::bytes::Bytes,
}
// @@protoc_insertion_point(module)
