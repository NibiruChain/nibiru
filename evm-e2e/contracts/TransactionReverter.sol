// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TransactionReverter {
    uint256 public value;

    error CustomRevertReason(uint256 providedValue);

    // Will try to set state and revert unconditionally
    function setAndRevert(uint256 newValue) public {
        value = newValue;
        revert("Transaction reverted after state change");
    }

    // Will revert with custom error
    function revertWithCustomError(uint256 newValue) public {
        value = newValue;
        revert CustomRevertReason(newValue);
    }

    // Will emit event and then revert
    event ValueUpdateAttempted(address sender, uint256 value);
    function emitAndRevert(uint256 newValue) public {
        emit ValueUpdateAttempted(msg.sender, newValue);
        value = newValue;
        revert("Reverted after event emission");
    }
}