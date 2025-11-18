// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {UserOperation, SIG_VALIDATION_FAILED} from "./UserOperation.sol";
import {IEntryPoint} from "./interfaces/IEntryPoint.sol";

/// @notice Minimal ERC-4337-style account secured by a P-256 pubkey (raw r,s signatures).
/// @dev Uses Nibiru RIP-7212 precompile at 0x...0100. Signature format: abi.encode(r,s).
contract PasskeyAccount {
    address public entryPoint;
    bytes32 public qx;
    bytes32 public qy;
    uint256 public nonce;
    bool private initialized;

    event Executed(address indexed target, uint256 value, bytes data);

    modifier onlyEntryPoint() {
        require(msg.sender == entryPoint, "not entrypoint");
        _;
    }

    function initialize(address _entryPoint, bytes32 _qx, bytes32 _qy) external {
        require(!initialized, "initialized");
        require(_entryPoint != address(0), "entrypoint=0");
        initialized = true;
        entryPoint = _entryPoint;
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
        if (!_verify(userOpHash, userOp.signature)) return SIG_VALIDATION_FAILED;
        nonce++;
        if (missingAccountFunds > 0) {
            (bool paid, ) = entryPoint.call{value: missingAccountFunds}("");
            require(paid, "pay failed");
        }
        return 0;
    }

    function execute(address to, uint256 value, bytes calldata data) external onlyEntryPoint {
        (bool ok, ) = to.call{value: value}(data);
        require(ok, "exec failed");
        emit Executed(to, value, data);
    }

    function _verify(bytes32 hash, bytes calldata signature) internal view returns (bool) {
        if (signature.length != 64) return false;
        (bytes32 r, bytes32 s) = abi.decode(signature, (bytes32, bytes32));
        bytes memory input = abi.encodePacked(hash, r, s, qx, qy);
        (bool ok, bytes memory out) = address(0x100).staticcall(input);
        return ok && out.length == 32 && out[31] == 0x01;
    }
}

/// @notice Simple factory deploying PasskeyAccount instances (no clones for simplicity).
contract PasskeyAccountFactory {
    address public immutable IMPLEMENTATION;
    address public immutable ENTRY_POINT;

    event AccountCreated(address indexed account, bytes32 qx, bytes32 qy);

    constructor(address _entryPoint) {
        ENTRY_POINT = _entryPoint;
        IMPLEMENTATION = address(new PasskeyAccount());
    }

    function createAccount(bytes32 _qx, bytes32 _qy) external returns (address account) {
        account = address(new PasskeyAccount());
        PasskeyAccount(payable(account)).initialize(ENTRY_POINT, _qx, _qy);
        emit AccountCreated(account, _qx, _qy);
    }
}
