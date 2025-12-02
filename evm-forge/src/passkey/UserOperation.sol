// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

/// @dev Lightweight copy of the ERC-4337 UserOperation struct to avoid pulling full AA deps in tests.
struct UserOperation {
    address sender;
    uint256 nonce;
    bytes initCode;
    bytes callData;
    uint256 callGasLimit;
    uint256 verificationGasLimit;
    uint256 preVerificationGas;
    uint256 maxFeePerGas;
    uint256 maxPriorityFeePerGas;
    bytes paymasterAndData;
    bytes signature;
}

uint256 constant SIG_VALIDATION_FAILED = 1;
