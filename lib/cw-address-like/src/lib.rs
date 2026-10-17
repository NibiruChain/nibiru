#![doc = include_str!(concat!(env!("CARGO_MANIFEST_DIR"), "/README.md"))]

use std::{
    fmt::{Debug, Display},
    hash::Hash,
};

/// Marks either `String` or `cosmwasm_std::Addr`.
///
/// String is used in unverified types, such as messages and query responses.
/// Addr is used in verified types, which are to be stored in blockchain state.
///
/// This trait is intended to be used as a generic in type definitions.
pub trait AddressLike:
    Clone + Debug + Display + PartialEq + Eq + PartialOrd + Ord + Hash
{
}

impl AddressLike for String {}
impl AddressLike for cosmwasm_std::Addr {}
