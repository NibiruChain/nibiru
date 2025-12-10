// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Clones} from "@openzeppelin/contracts/proxy/Clones.sol";
import {UserOperation, SIG_VALIDATION_FAILED} from "./UserOperation.sol";
import {IEntryPoint} from "./interfaces/IEntryPoint.sol";

/// @notice Minimal ERC-4337-style account secured by a P-256 pubkey (raw r,s signatures).
/// @dev Uses Nibiru RIP-7212 precompile at 0x...0100. Signature format: abi.encode(r,s).
contract PasskeyAccount {
    IEntryPoint public entryPoint;
    bytes32 public qx;
    bytes32 public qy;
    uint256 public nonce;
    bool private initialized;

    event Executed(address indexed target, uint256 value, bytes data);

    modifier onlyEntryPoint() {
        require(msg.sender == address(entryPoint), "not entrypoint");
        _;
    }

    function initialize(address _entryPoint, bytes32 _qx, bytes32 _qy) external {
        require(!initialized, "initialized");
        require(_entryPoint != address(0), "entrypoint=0");
        initialized = true;
        entryPoint = IEntryPoint(_entryPoint);
        qx = _qx;
        qy = _qy;
    }

    receive() external payable {}

    function validateUserOp(
        UserOperation calldata userOp,
        bytes32 userOpHash,
        uint256 missingAccountFunds
    ) external onlyEntryPoint returns (uint256 validationData) {
        if (userOp.nonce != nonce) return SIG_VALIDATION_FAILED;
        uint256 verified = _verify(userOpHash, userOp.signature);
        if (verified != 1) return SIG_VALIDATION_FAILED;
        nonce++;
        if (missingAccountFunds > 0) {
            entryPoint.depositTo{value: missingAccountFunds}(address(this));
        }
        return 0;
    }

    function execute(address to, uint256 value, bytes calldata data) external onlyEntryPoint {
        (bool ok, ) = to.call{value: value}(data);
        require(ok, "exec failed");
        emit Executed(to, value, data);
    }

    function _verify(bytes32 userOpHash, bytes calldata signature) internal view returns (uint256) {
        bytes32 digest;
        bytes32 r;
        bytes32 s;

        // For Node/EVM tests we accept compact r,s signatures over userOpHash.
        if (signature.length == 64) {
            (r, s) = abi.decode(signature, (bytes32, bytes32));
            digest = sha256(abi.encodePacked(userOpHash));
        } else {
            // WebAuthn signatures are over sha256(authenticatorData || sha256(clientDataJSON)).
            // Signature payload layout: abi.encode(authenticatorData, clientDataJSON, r, s)
            (bytes memory authData, bytes memory clientDataJSON, bytes32 rFull, bytes32 sFull) =
                abi.decode(signature, (bytes, bytes, bytes32, bytes32));
            r = rFull;
            s = sFull;
            digest = sha256(abi.encodePacked(authData, sha256(clientDataJSON)));
        }

        bytes memory input = abi.encodePacked(digest, r, s, qx, qy);
        (bool ok, bytes memory out) = address(0x100).staticcall(input);
        if (!ok || out.length != 32) {
            return 0;
        }
        return uint256(bytes32(out));
    }
}

/// @notice Factory deploying cheap PasskeyAccount minimal-proxy clones.
contract PasskeyAccountFactory {
    address public immutable IMPLEMENTATION;
    address public immutable ENTRY_POINT;

    event AccountCreated(address indexed account, bytes32 qx, bytes32 qy, bytes32 salt);

    constructor(address _entryPoint) {
        ENTRY_POINT = _entryPoint;
        IMPLEMENTATION = address(new PasskeyAccount());
    }

    function createAccount(bytes32 _qx, bytes32 _qy) external returns (address account) {
        bytes32 salt = _salt(_qx, _qy);
        account = Clones.cloneDeterministic(IMPLEMENTATION, salt);
        PasskeyAccount(payable(account)).initialize(ENTRY_POINT, _qx, _qy);
        emit AccountCreated(account, _qx, _qy, salt);
    }

    function accountAddress(bytes32 _qx, bytes32 _qy) external view returns (address predicted) {
        bytes32 salt = _salt(_qx, _qy);
        predicted = Clones.predictDeterministicAddress(IMPLEMENTATION, salt, address(this));
    }

    function _salt(bytes32 _qx, bytes32 _qy) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(_qx, _qy));
    }
}
