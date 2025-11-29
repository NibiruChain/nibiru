// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import "forge-std/src/Script.sol";

import { PasskeyAccountFactory } from "../src/passkey/PasskeyAccount.sol";
import { IEntryPoint } from "../src/passkey/interfaces/IEntryPoint.sol";

/// @notice Deploys a PasskeyAccountFactory and optionally creates the first account.
/// @dev Required env vars:
/// - ENTRY_POINT: address of deployed ERC-4337 EntryPoint on Nibiru EVM
///
/// Optional env vars (to immediately create an account):
/// - QX, QY: bytes32 affine coords for the user's P-256 pubkey
/// - SALT: bytes32 salt for deterministic cloning (default: 0)
contract DeployPasskeyFactory is Script {
    function run() external {
        address entryPoint = vm.envAddress("ENTRY_POINT");
        bytes32 salt = _envOrBytes32("SALT", bytes32(0));

        vm.startBroadcast();

        PasskeyAccountFactory factory = new PasskeyAccountFactory(IEntryPoint(entryPoint));
        console2.log("PasskeyAccountFactory", address(factory));

        (bool hasQx, bytes32 qx) = _tryEnvBytes32("QX");
        (bool hasQy, bytes32 qy) = _tryEnvBytes32("QY");
        if (hasQx && hasQy) {
            address account = factory.createAccount(qx, qy, salt);
            console2.log("PasskeyAccount", account);
        } else {
            console2.log("Skipping account creation (QX/QY not provided)");
        }

        vm.stopBroadcast();
    }

    function _tryEnvBytes32(string memory key) internal returns (bool ok, bytes32 value) {
        try vm.envBytes32(key) returns (bytes32 v) {
            return (true, v);
        } catch {
            return (false, bytes32(0));
        }
    }

    function _envOrBytes32(string memory key, bytes32 defaultValue) internal returns (bytes32) {
        (bool ok, bytes32 v) = _tryEnvBytes32(key);
        return ok ? v : defaultValue;
    }
}
