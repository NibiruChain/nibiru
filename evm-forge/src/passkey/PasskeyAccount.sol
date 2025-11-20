// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { Clones } from "@openzeppelin/contracts/proxy/Clones.sol";

import { P256Precompile } from "./P256.sol";
import { UserOperation, SIG_VALIDATION_FAILED } from "./UserOperation.sol";
import { IEntryPoint } from "./interfaces/IEntryPoint.sol";

/// @notice Minimal ERC-4337-style account secured by a stored P-256 public key.
/// @dev Signature format is compact: signature = abi.encode(r, s) with both 32-byte.
contract PasskeyAccount {
    using Address for address;

    error Initialized();
    error NotEntryPoint();
    error InvalidSignature();
    error InvalidNonce();

    IEntryPoint public entryPoint;
    bytes32 public qx;
    bytes32 public qy;
    uint256 public nonce;
    bool private initialized;

    modifier onlyEntryPoint() {
        if (msg.sender != address(entryPoint)) revert NotEntryPoint();
        _;
    }

    function initialize(IEntryPoint _entryPoint, bytes32 _qx, bytes32 _qy) external {
        if (initialized) revert Initialized();
        if (address(_entryPoint) == address(0)) revert NotEntryPoint();
        initialized = true;
        entryPoint = _entryPoint;
        qx = _qx;
        qy = _qy;
    }

    receive() external payable { }

    /// @notice ERC-4337 validation hook. Returns 0 on success, SIG_VALIDATION_FAILED on bad sig/nonce.
    function validateUserOp(
        UserOperation calldata userOp,
        bytes32 userOpHash,
        uint256 missingAccountFunds
    )
        external
        onlyEntryPoint
        returns (uint256 validationData)
    {
        if (userOp.nonce != nonce) {
            return SIG_VALIDATION_FAILED;
        }
        if (!_verifySignature(userOpHash, userOp.signature)) {
            return SIG_VALIDATION_FAILED;
        }
        nonce++;

        if (missingAccountFunds > 0) {
            entryPoint.depositTo{ value: missingAccountFunds }(address(this));
        }
        return 0;
    }

    /// @notice Execute a call from the EntryPoint.
    function execute(address to, uint256 value, bytes calldata data) external onlyEntryPoint {
        to.functionCallWithValue(data, value);
    }

    function _verifySignature(bytes32 userOpHash, bytes calldata signature) internal view returns (bool) {
        if (signature.length != 64) return false;
        (bytes32 r, bytes32 s) = abi.decode(signature, (bytes32, bytes32));
        bytes32 digest = sha256(abi.encodePacked(userOpHash));
        return P256Precompile.verify(digest, r, s, qx, qy);
    }

}

/// @notice Simple clone factory for PasskeyAccount instances.
contract PasskeyAccountFactory {
    address public immutable IMPLEMENTATION;
    IEntryPoint public immutable ENTRY_POINT;

    event AccountCreated(address indexed account, bytes32 qx, bytes32 qy, bytes32 salt);

    constructor(IEntryPoint _entryPoint) {
        IMPLEMENTATION = address(new PasskeyAccount());
        ENTRY_POINT = _entryPoint;
    }

    function createAccount(bytes32 _qx, bytes32 _qy, bytes32 salt) external returns (address account) {
        account = Clones.cloneDeterministic(IMPLEMENTATION, salt);
        PasskeyAccount(payable(account)).initialize(ENTRY_POINT, _qx, _qy);
        emit AccountCreated(account, _qx, _qy, salt);
    }

    function accountAddress(bytes32 salt) external view returns (address predicted) {
        predicted = Clones.predictDeterministicAddress(IMPLEMENTATION, salt, address(this));
    }
}
