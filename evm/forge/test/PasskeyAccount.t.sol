// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import "forge-std/src/Test.sol";

import { PasskeyAccount, PasskeyAccountFactory } from "../src/passkey/PasskeyAccount.sol";
import { UserOperation, SIG_VALIDATION_FAILED } from "../src/passkey/UserOperation.sol";
import { IEntryPoint } from "../src/passkey/interfaces/IEntryPoint.sol";

contract PasskeyAccountTest is Test {
    // Deterministic test vector derived from Go helper (d = 1 on P-256).
    bytes32 private constant HASH = 0xab4c2965c23ed3396c0e8867a1074abb369bd6ab0eba3c3b59c3cedaecb2f91d;
    bytes32 private constant R = 0x4dfc47f23ab84ee2d6e733ae536124b8af901561670cb1831172d3e1904e1fc2;
    bytes32 private constant S = 0x58ab39a42f448954edbc1a085c47cf6420cbe37ed193eaa5bd00d53c77f17ccb;
    bytes32 private constant QX = 0x6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296;
    bytes32 private constant QY = 0x4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5;

    address private constant P256_PRECOMPILE = address(0x100);

    EntryPointStub private entryPoint;
    PasskeyAccountFactory private factory;
    PasskeyAccount private account;

    function setUp() public {
        entryPoint = new EntryPointStub();
        factory = new PasskeyAccountFactory(entryPoint);
        account = PasskeyAccount(payable(factory.createAccount(QX, QY, bytes32("salt"))));
    }

    function testValidateUserOpSuccessPaysMissingFunds() public {
        vm.deal(address(account), 1 ether);
        _mockP256(HASH, true);

        UserOperation memory op = _defaultOp();
        uint256 missing = 1 wei;

        uint256 res = _callValidate(op, HASH, missing);
        assertEq(res, 0, "validation should succeed");
        assertEq(account.nonce(), 1, "nonce increments");
        assertEq(address(entryPoint).balance, missing, "entrypoint paid");
    }

    function testValidateUserOpRejectsBadSignature() public {
        _mockP256(HASH, true);

        UserOperation memory op = _defaultOp();

        // Bad hash -> mock not triggered, so verification fails.
        uint256 res = _callValidate(op, HASH ^ bytes32(uint256(1)), 0);
        assertEq(res, SIG_VALIDATION_FAILED, "bad hash rejected");
        assertEq(account.nonce(), 0, "nonce should not increment");

        // Bad sig length.
        op.signature = hex"01";
        res = _callValidate(op, HASH, 0);
        assertEq(res, SIG_VALIDATION_FAILED, "bad sig length rejected");
    }

    function testValidateUserOpRejectsReplay() public {
        vm.deal(address(account), 1 ether);
        _mockP256(HASH, true);

        UserOperation memory op = _defaultOp();
        assertEq(_callValidate(op, HASH, 0), 0, "first call ok");
        assertEq(_callValidate(op, HASH, 0), SIG_VALIDATION_FAILED, "replay fails nonce");
    }

    function testExecuteOnlyEntryPoint() public {
        Target target = new Target();
        bytes memory callData = abi.encodeWithSignature("ping()");

        vm.expectRevert(PasskeyAccount.NotEntryPoint.selector);
        account.execute(address(target), 0, callData);

        vm.prank(address(entryPoint));
        account.execute(address(target), 0, callData);
        assertTrue(target.pinged(), "target should be pinged");
    }

    function _callValidate(
        UserOperation memory op,
        bytes32 userOpHash,
        uint256 missingFunds
    )
        internal
        returns (uint256)
    {
        vm.prank(address(entryPoint));
        return account.validateUserOp(op, userOpHash, missingFunds);
    }

    function _mockP256(bytes32 hash, bool valid) internal {
        bytes memory input = abi.encodePacked(hash, R, S, QX, QY);
        bytes memory output = new bytes(32);
        if (valid) {
            output[31] = 0x01;
        }
        vm.mockCall(P256_PRECOMPILE, input, output);
    }

    function _defaultOp() internal pure returns (UserOperation memory op) {
        op = UserOperation({
            sender: address(0), // unused in this validation path
            nonce: 0,
            initCode: "",
            callData: "",
            callGasLimit: 0,
            verificationGasLimit: 0,
            preVerificationGas: 0,
            maxFeePerGas: 0,
            maxPriorityFeePerGas: 0,
            paymasterAndData: "",
            signature: abi.encode(R, S)
        });
    }
}

contract EntryPointStub is IEntryPoint {
    function balanceOf(address) external pure returns (uint256) {
        return 0;
    }

    function handleOps(UserOperation[] calldata, address payable) external pure { }

    receive() external payable { }
}

contract Target {
    bool public pinged;

    function ping() external {
        pinged = true;
    }
}
