// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract GasIntensiveERC20 is ERC20 {
    constructor() ERC20("GasIntensiveToken", "GIT") {
        _mint(msg.sender, 1000000 * 10**18); // Mint 1 million tokens to deployer
    }

//    function name() public view virtual override returns (string memory) {
//        string memory actualName = super.name();
//        _gasIntensiveOperation();
//        return actualName;
//    }
//
//    function balanceOf(address account) public view virtual override returns (uint256) {
//        uint256 actualBalance = super.balanceOf(account);
//        _gasIntensiveOperation();
//        return actualBalance;
//    }

    function transfer(address recipient, uint256 amount) public virtual override returns (bool) {
        _gasIntensiveOperation();
        return super.transfer(recipient, amount);
    }

    // Gas-intensive operation to simulate high computational cost
    function _gasIntensiveOperation() internal pure {
        uint256 result = 1;
        for (uint256 i = 0; i < 1000000; i++) {
            result = result * 2 + 1;
            result = result / 2;
            result = result ^ (result << 1);
            result = result & 0xFFFFFFFFFFFFFFFF;
        }
        // The result is not used, ensuring the compiler doesn't optimize this away
        assert(result != 0);
    }
}