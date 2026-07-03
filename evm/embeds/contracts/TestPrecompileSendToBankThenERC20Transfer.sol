// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "./IFunToken.sol";

contract TestPrecompileSendToBankThenERC20Transfer {
    IERC20 public erc20;
    string public recipient;

    constructor(address _erc20, string memory _recipient) {
        erc20 = IERC20(_erc20);
        recipient = _recipient;
    }

    function attack() public {
        // transfer this contract's entire balance to the recipient
        uint balance = erc20.balanceOf(address(this));
        // sendToBank should reduce balance to zero
        FUNTOKEN_PRECOMPILE.sendToBank(address(erc20), balance, recipient);

        // this call should fail because of the balance is zero
        erc20.transfer(0x000000000000000000000000000000000000dEaD, 1);
    }
}
