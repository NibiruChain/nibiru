// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestERC20TransferWithFee is ERC20 {
    uint256 constant FEE_PERCENTAGE = 10;

    constructor(string memory name, string memory symbol)
    ERC20(name, symbol) {
        _mint(msg.sender, 1000);
    }

    function transfer(address to, uint256 amount) public virtual override returns (bool) {
        address owner = _msgSender();
        require(amount > 0, "Transfer amount must be greater than zero");

        uint256 fee = (amount * FEE_PERCENTAGE) / 100;
        uint256 recipientAmount = amount - fee;

        _transfer(owner, address(this), fee);
        _transfer(owner, to, recipientAmount);

        return true;
    }
}
