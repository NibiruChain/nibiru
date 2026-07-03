// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract EventsEmitter {
    event TestEvent(address indexed sender, uint256 value);

    function emitEvent(uint256 value) public {
        emit TestEvent(msg.sender, value);
    }
}