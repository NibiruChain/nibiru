// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

/// @title P256Precompile
/// @notice Thin wrapper around the RIP-7212 style precompile living at 0x000...0100.
library P256Precompile {
    address internal constant PRECOMPILE = 0x0000000000000000000000000000000000000100;
    uint256 internal constant INPUT_LEN = 160; // 32 * 5
    uint256 internal constant OUTPUT_LEN = 32;

    /// @dev Returns true if the precompile reports a valid signature.
    function verify(bytes32 hash, bytes32 r, bytes32 s, bytes32 qx, bytes32 qy) internal view returns (bool) {
        bytes memory input = abi.encodePacked(hash, r, s, qx, qy);
        if (input.length != INPUT_LEN) {
            return false;
        }

        (bool ok, bytes memory out) = PRECOMPILE.staticcall(input);
        if (!ok || out.length != OUTPUT_LEN) {
            return false;
        }
        return out[OUTPUT_LEN - 1] == 0x01;
    }
}
