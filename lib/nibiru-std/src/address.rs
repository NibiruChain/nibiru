//! Address conversion utilities for Nibiru

use bech32::{self, FromBase32, ToBase32};

use crate::errors::{NibiruError, NibiruResult};

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
}
