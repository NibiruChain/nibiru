// contracts/TestERC20.sol
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract TestRandom {

    function getRandom() public view returns (uint256) {
        return block.prevrandao;
    }
}
