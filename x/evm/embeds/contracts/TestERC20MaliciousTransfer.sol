// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestERC20MaliciousTransfer is ERC20 {
    constructor(string memory name, string memory symbol, uint8 decimals_)
    ERC20(name, symbol) {
        _mint(msg.sender, 1000000 * 10**18);
    }

    function transfer(address recipient, uint256 amount) public virtual override returns (bool) {
        _gasIntensiveOperation();
        return super.transfer(recipient, amount);
    }

    // Gas-intensive operation to simulate high computational cost
    function _gasIntensiveOperation() internal pure {
        uint256 result = 1;
        for (uint256 i = 0; i < 100000; i++) {
            result = result * 2 + 1;
            result = result / 2;
            result = result ^ (result << 1);
            result = result & 0xFFFFFFFFFFFFFFFF;
        }
        // The result is not used, ensuring the compiler doesn't optimize this away
        assert(result != 0);
    }
}