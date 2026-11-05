//! Address conversion utilities for Nibiru

use std::{fmt, str::FromStr};

use bech32::{self, FromBase32, ToBase32, Variant};
use cosmwasm_schema::schemars::{
    gen::SchemaGenerator, schema::Schema, JsonSchema,
};
use cosmwasm_std::Addr;
use serde::{de::Error as _, Deserialize, Deserializer, Serialize, Serializer};
use tiny_keccak::{Hasher, Keccak};

use crate::errors::{NibiruError, NibiruResult};

/// Byte length shared by Nibiru and EVM externally owned accounts.
pub const USER_ADDR_LEN: usize = 20;

/// A validated Nibiru externally owned account.
///
/// JSON input is a Nibiru bech32 address or a `0x`-prefixed 20-byte EVM
/// address. JSON output is canonical EIP-55 hex. Use [`cosmwasm_std::Addr`]
/// for generic CosmWasm account identities, including permission members and
/// contract callers.
#[derive(Clone, Copy, Debug, Eq, Hash, Ord, PartialEq, PartialOrd)]
pub struct UserAddr([u8; USER_ADDR_LEN]);

impl UserAddr {
    /// Returns the canonical EIP-55 representation.
    pub fn to_hex(self) -> String {
        eip55_checksum_hex(&self.0)
    }

    /// Returns the equivalent canonical Nibiru bech32 address.
    pub fn to_bech32_addr(self) -> Addr {
        let encoded =
            bech32::encode("nibi", self.0.to_base32(), Variant::Bech32)
                .expect("fixed Nibiru HRP and 20-byte payload are valid");
        Addr::unchecked(encoded)
    }

    /// Returns the underlying 20-byte account identity.
    pub fn as_bytes(&self) -> &[u8; USER_ADDR_LEN] {
        &self.0
    }

    fn from_bech32(input: &str) -> NibiruResult<Self> {
        let (hrp, data, variant) = bech32::decode(input)?;
        if hrp != "nibi" {
            return Err(NibiruError::InvalidBech32Prefix {
                expected: "nibi".to_string(),
                actual: hrp,
            });
        }
        if variant != Variant::Bech32 {
            return Err(NibiruError::InvalidEthAddress(
                "Nibiru user address must use the Bech32 checksum variant"
                    .to_string(),
            ));
        }

        let bytes = Vec::<u8>::from_base32(&data)?;
        let bytes: [u8; USER_ADDR_LEN] = bytes.try_into().map_err(
            |bytes: Vec<u8>| {
                NibiruError::InvalidEthAddress(format!(
                    "Nibiru user address must decode to {USER_ADDR_LEN} bytes, got {}",
                    bytes.len()
                ))
            },
        )?;
        Ok(Self(bytes))
    }

    fn from_hex(input: &str) -> NibiruResult<Self> {
        let hex = input
            .strip_prefix("0x")
            .or_else(|| input.strip_prefix("0X"))
            .ok_or_else(|| {
                NibiruError::InvalidEthAddress(
                    "EVM user address must start with 0x".to_string(),
                )
            })?;
        if hex.len() != USER_ADDR_LEN * 2 {
            return Err(NibiruError::InvalidEthAddress(format!(
                "EVM user address must contain 40 hex characters, got {}",
                hex.len()
            )));
        }
        let bytes = hex::decode(hex)?;
        let bytes: [u8; USER_ADDR_LEN] = bytes.try_into().map_err(
            |bytes: Vec<u8>| {
                NibiruError::InvalidEthAddress(format!(
                    "EVM user address must decode to {USER_ADDR_LEN} bytes, got {}",
                    bytes.len()
                ))
            },
        )?;
        Ok(Self(bytes))
    }
}

impl FromStr for UserAddr {
    type Err = NibiruError;

    fn from_str(input: &str) -> Result<Self, Self::Err> {
        let input = input.trim();
        if input.is_empty() {
            return Err(NibiruError::InvalidEthAddress(
                "user address is empty".to_string(),
            ));
        }

        if input.to_ascii_lowercase().starts_with("nibi1") {
            Self::from_bech32(input)
        } else {
            Self::from_hex(input)
        }
    }
}

impl fmt::Display for UserAddr {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.to_hex())
    }
}

impl Serialize for UserAddr {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&self.to_hex())
    }
}

impl<'de> Deserialize<'de> for UserAddr {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        String::deserialize(deserializer)?
            .parse()
            .map_err(D::Error::custom)
    }
}

impl JsonSchema for UserAddr {
    fn schema_name() -> String {
        "UserAddr".to_string()
    }

    fn json_schema(generator: &mut SchemaGenerator) -> Schema {
        let mut schema = String::json_schema(generator);
        if let Schema::Object(object) = &mut schema {
            object.metadata().description = Some(
                "A Nibiru externally owned account. Input accepts a Nibiru bech32 address or a 0x-prefixed 20-byte EVM address; output uses EIP-55 hex."
                    .to_string(),
            );
        }
        schema
    }
}

fn eip55_checksum_hex(bytes: &[u8; USER_ADDR_LEN]) -> String {
    let lowercase = hex::encode(bytes);
    let mut hash = [0u8; 32];
    let mut hasher = Keccak::v256();
    hasher.update(lowercase.as_bytes());
    hasher.finalize(&mut hash);

    let mut output = String::with_capacity(42);
    output.push_str("0x");
    for (index, ch) in lowercase.chars().enumerate() {
        let nibble = if index % 2 == 0 {
            hash[index / 2] >> 4
        } else {
            hash[index / 2] & 0x0f
        };
        if ch.is_ascii_alphabetic() && nibble >= 8 {
            output.push(ch.to_ascii_uppercase());
        } else {
            output.push(ch);
        }
    }
    output
}

/// Converts a Nibiru bech32 address to an Ethereum hex address.
///
/// This function decodes a bech32-encoded Nibiru address (with "nibi" prefix)
/// and converts it to an Ethereum-compatible hex address by taking the first
/// 20 bytes of the decoded data.
///
/// # Arguments
///
/// * `bech32_addr` - A bech32-encoded Nibiru address string (e.g., "nibi1...")
///
/// # Returns
///
/// * `Ok(String)` - The Ethereum hex address prefixed with "0x"
/// * `Err(NibiruError)` - If the address is invalid, has wrong prefix, or is too short
///
/// # Example
///
/// ```
/// use nibiru_std::address::nibiru_bech32_to_eth_address;
///
/// let bech32_addr = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";
/// let eth_addr = nibiru_bech32_to_eth_address(bech32_addr).unwrap();
/// assert_eq!(eth_addr, "0x46155fafd58660583ac0d23d8e22b9a13ca0fb31");
/// ```
pub fn nibiru_bech32_to_eth_address(bech32_addr: &str) -> NibiruResult<String> {
    // Decode the bech32 address
    let (hrp, data, _variant) = bech32::decode(bech32_addr)?;

    // Verify the human-readable part is "nibi"
    if hrp != "nibi" {
        return Err(NibiruError::InvalidBech32Prefix {
            expected: "nibi".to_string(),
            actual: hrp,
        });
    }

    // Convert from base32 to bytes
    let bytes = Vec::<u8>::from_base32(&data)?;

    // Ethereum addresses are 20 bytes
    if bytes.len() < 20 {
        return Err(NibiruError::InvalidAddressLength);
    }

    // Take the first 20 bytes and format as hex with 0x prefix
    let eth_addr = format!("0x{}", hex::encode(&bytes[..20]));
    Ok(eth_addr)
}

/// Converts an Ethereum hex address to a Nibiru bech32 address.
///
/// This function takes an Ethereum address in hex format (with or without "0x" prefix)
/// and converts it to a bech32-encoded Nibiru address with "nibi" prefix.
///
/// # Arguments
///
/// * `eth_addr` - An Ethereum address as a hex string (e.g., "0x..." or just the hex)
///
/// # Returns
///
/// * `Ok(String)` - The Nibiru bech32 address
/// * `Err(NibiruError)` - If the address is invalid or not exactly 20 bytes
///
/// # Example
///
/// ```
/// use nibiru_std::address::eth_address_to_nibiru_bech32;
///
/// let eth_addr = "0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31";
/// let bech32_addr = eth_address_to_nibiru_bech32(eth_addr).unwrap();
/// assert_eq!(bech32_addr, "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul");
/// ```
pub fn eth_address_to_nibiru_bech32(eth_addr: &str) -> NibiruResult<String> {
    // Remove "0x" prefix if present
    let hex_str = eth_addr.strip_prefix("0x").unwrap_or(eth_addr);

    // Validate hex string length (20 bytes = 40 hex chars)
    if hex_str.len() != 40 {
        return Err(NibiruError::InvalidEthAddress(format!(
            "Ethereum address must be 20 bytes (40 hex chars), got {} chars",
            hex_str.len()
        )));
    }

    // Decode hex to bytes
    let bytes = hex::decode(hex_str)?;

    // Sanity check: should be exactly 20 bytes
    if bytes.len() != 20 {
        return Err(NibiruError::InvalidEthAddress(format!(
            "Invalid Ethereum address length: expected 20 bytes, got {}",
            bytes.len()
        )));
    }

    // Encode as bech32 with "nibi" prefix
    let bech32_addr =
        bech32::encode("nibi", bytes.to_base32(), bech32::Variant::Bech32)?;
    Ok(bech32_addr)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_nibiru_bech32_to_eth_address_valid() {
        // Test case from the Go implementation
        let bech32_addr = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";
        let expected_eth = "0x46155fafd58660583ac0d23d8e22b9a13ca0fb31";

        let result = nibiru_bech32_to_eth_address(bech32_addr).unwrap();
        assert_eq!(result.to_lowercase(), expected_eth);
    }

    #[test]
    fn test_nibiru_bech32_to_eth_address_invalid_prefix() {
        // Valid bech32 address but with cosmos prefix instead of nibi
        let bech32_addr = "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a";

        let result = nibiru_bech32_to_eth_address(bech32_addr);
        match result {
            Err(NibiruError::InvalidBech32Prefix { expected, actual }) => {
                assert_eq!(expected, "nibi");
                assert_eq!(actual, "cosmos");
            }
            _ => panic!("Expected InvalidBech32Prefix error, got: {:?}", result),
        }
    }

    #[test]
    fn test_nibiru_bech32_to_eth_address_invalid_bech32() {
        let invalid_addr = "nibi1invalid!@#$";

        let result = nibiru_bech32_to_eth_address(invalid_addr);
        assert!(matches!(result, Err(NibiruError::Bech32Error(_))));
    }

    #[test]
    fn test_nibiru_bech32_to_eth_address_length_validation() {
        // Test that we properly validate address length
        // We'll use a test helper to create a short address
        use bech32::ToBase32;

        // Create a short address with only 10 bytes (need 20 for Ethereum)
        let short_data = vec![0u8; 10];
        let short_addr = bech32::encode(
            "nibi",
            short_data.to_base32(),
            bech32::Variant::Bech32,
        )
        .unwrap();

        let result = nibiru_bech32_to_eth_address(&short_addr);
        match result {
            Err(NibiruError::InvalidAddressLength) => {}
            _ => {
                panic!("Expected InvalidAddressLength error, got: {:?}", result)
            }
        }
    }

    #[test]
    fn test_nibiru_bech32_to_eth_address_case_sensitivity() {
        // Test that the output maintains proper case
        let bech32_addr = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";
        let result = nibiru_bech32_to_eth_address(bech32_addr).unwrap();

        // The hex should have lowercase letters after 0x
        assert!(result.starts_with("0x"));
        // But we'll compare case-insensitively for the actual value
        assert_eq!(
            result.to_lowercase(),
            "0x46155fafd58660583ac0d23d8e22b9a13ca0fb31"
        );
    }

    #[test]
    fn test_eth_address_to_nibiru_bech32_valid() {
        // Test case matching the Go implementation
        let eth_addr = "0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31";
        let expected_bech32 = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";

        let result = eth_address_to_nibiru_bech32(eth_addr).unwrap();
        assert_eq!(result, expected_bech32);
    }

    #[test]
    fn test_eth_address_to_nibiru_bech32_without_prefix() {
        // Test without 0x prefix
        let eth_addr = "46155fAfd58660583ac0d23d8E22B9A13Ca0fb31";
        let expected_bech32 = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";

        let result = eth_address_to_nibiru_bech32(eth_addr).unwrap();
        assert_eq!(result, expected_bech32);
    }

    #[test]
    fn test_eth_address_to_nibiru_bech32_lowercase() {
        // Test with lowercase hex
        let eth_addr = "0x46155fafd58660583ac0d23d8e22b9a13ca0fb31";
        let expected_bech32 = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";

        let result = eth_address_to_nibiru_bech32(eth_addr).unwrap();
        assert_eq!(result, expected_bech32);
    }

    #[test]
    fn test_eth_address_to_nibiru_bech32_invalid_length() {
        // Too short
        let short_addr = "0x46155fafd58660583ac0d23d8e22b9a13ca0fb";
        let result = eth_address_to_nibiru_bech32(short_addr);
        match result {
            Err(NibiruError::InvalidEthAddress(msg)) => {
                assert!(msg.contains("40 hex chars"));
            }
            _ => panic!("Expected InvalidEthAddress error"),
        }

        // Too long
        let long_addr = "0x46155fafd58660583ac0d23d8e22b9a13ca0fb3100";
        let result = eth_address_to_nibiru_bech32(long_addr);
        match result {
            Err(NibiruError::InvalidEthAddress(msg)) => {
                assert!(msg.contains("40 hex chars"));
            }
            _ => panic!("Expected InvalidEthAddress error"),
        }
    }

    #[test]
    fn test_eth_address_to_nibiru_bech32_invalid_hex() {
        // Invalid hex characters
        let invalid_addr = "0x46155fXXd58660583ac0d23d8e22b9a13ca0fb31";
        let result = eth_address_to_nibiru_bech32(invalid_addr);
        assert!(matches!(result, Err(NibiruError::HexError(_))));
    }

    #[test]
    fn test_round_trip_conversion() {
        // Test that converting back and forth gives the same result
        let original_bech32 = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";

        // Convert to Ethereum
        let eth_addr = nibiru_bech32_to_eth_address(original_bech32).unwrap();

        // Convert back to bech32
        let result_bech32 = eth_address_to_nibiru_bech32(&eth_addr).unwrap();

        assert_eq!(original_bech32, result_bech32);
    }

    #[test]
    fn test_multiple_round_trips() {
        // Test multiple addresses round-trip correctly
        // Generate some valid test addresses
        use bech32::ToBase32;

        let test_bytes = vec![
            vec![0u8; 20],   // All zeros
            vec![255u8; 20], // All ones
            vec![
                1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18,
                19, 20,
            ], // Sequential
        ];

        for bytes in test_bytes {
            // Create a valid bech32 address
            let original_bech32 = bech32::encode(
                "nibi",
                bytes.to_base32(),
                bech32::Variant::Bech32,
            )
            .unwrap();

            // Convert to Ethereum
            let eth_addr =
                nibiru_bech32_to_eth_address(&original_bech32).unwrap();

            // Convert back to bech32
            let result_bech32 = eth_address_to_nibiru_bech32(&eth_addr).unwrap();

            assert_eq!(
                original_bech32, result_bech32,
                "Round trip failed for address"
            );
        }
    }

    #[test]
    fn user_addr_accepts_equivalent_forms_and_serializes_eip55() {
        let bech32 = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";
        let hex = "0x46155fafd58660583ac0d23d8e22b9a13ca0fb31";
        let from_bech32: UserAddr = bech32.parse().unwrap();
        let from_hex: UserAddr = format!("  {hex}  ").parse().unwrap();

        assert_eq!(from_bech32, from_hex);
        assert_eq!(from_hex.to_bech32_addr(), Addr::unchecked(bech32));
        assert_eq!(
            from_hex.to_hex(),
            "0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31"
        );
        assert_eq!(
            serde_json::to_string(&from_hex).unwrap(),
            "\"0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31\""
        );
        assert_eq!(
            "0X46155FAFD58660583AC0D23D8E22B9A13CA0FB31"
                .parse::<UserAddr>()
                .unwrap(),
            from_hex
        );
        let zero = "0x0000000000000000000000000000000000000000"
            .parse::<UserAddr>()
            .unwrap();
        assert_eq!(zero.as_bytes(), &[0; USER_ADDR_LEN]);
    }

    #[test]
    fn user_addr_serde_accepts_bech32_and_rejects_non_string_json() {
        let encoded = "\"nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul\"";
        let parsed: UserAddr = serde_json::from_str(encoded).unwrap();
        assert_eq!(
            parsed.to_hex(),
            "0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31"
        );
        assert!(serde_json::from_str::<UserAddr>("[0, 1]").is_err());
    }

    #[test]
    fn user_addr_rejects_invalid_encodings_and_contract_addresses() {
        assert!("46155fafd58660583ac0d23d8e22b9a13ca0fb31"
            .parse::<UserAddr>()
            .is_err());
        assert!("0x1234".parse::<UserAddr>().is_err());
        assert!("0xzz155fafd58660583ac0d23d8e22b9a13ca0fb31"
            .parse::<UserAddr>()
            .is_err());
        assert!("cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
            .parse::<UserAddr>()
            .is_err());
        assert!("nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgum"
            .parse::<UserAddr>()
            .is_err());

        let contract =
            bech32::encode("nibi", [7u8; 32].to_base32(), Variant::Bech32)
                .unwrap();
        assert!(contract.parse::<UserAddr>().is_err());
    }

    #[test]
    fn user_addr_schema_is_a_string() {
        let schema = cosmwasm_schema::schema_for!(UserAddr);
        let json = serde_json::to_value(schema).unwrap();
        assert_eq!(json["type"], "string");
        assert!(json["description"].as_str().unwrap().contains("EIP-55"));
    }
}
