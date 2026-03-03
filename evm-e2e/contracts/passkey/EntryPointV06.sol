// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.8.24;

import "@account-abstraction/contracts/core/EntryPoint.sol";

/// @notice Thin wrapper that tags EntryPoint v0.6 with a version string for bundlers that require it.
contract EntryPointV06 is EntryPoint {
    function entryPointVersion() external pure returns (string memory) {
        return "0.6.0";
    }
}
