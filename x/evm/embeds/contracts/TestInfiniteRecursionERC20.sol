// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IFunToken.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestInfiniteRecursionERC20 is ERC20 {
    constructor(string memory name, string memory symbol, uint8 decimals_)
    ERC20(name, symbol) {
        _mint(msg.sender, 1000000 * 10**18);
    }

    function balanceOf(address who) public view virtual override returns (uint256) {
        // recurse through funtoken.balance(who, address(this))
        address(FUNTOKEN_PRECOMPILE_ADDRESS).staticcall(
            abi.encodeWithSignature(
                "balance(address,address)",
                who,
                address(this))
        );
        return 0;
    }

    function transfer(address to, uint256 amount) public override returns (bool) {
        // recurse through funtoken sendToBank
        FUNTOKEN_PRECOMPILE.sendToBank(
            address(this),
            amount,
            "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl" // does not matter, it's not reached
        );
        return true;
    }

    function attackBalance() public {
        balanceOf(address(0));
    }

    function attackTransfer() public {
        transfer(address(0), 1);
    }
}
