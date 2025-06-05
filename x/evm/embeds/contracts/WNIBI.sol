// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title WNIBI
/// @notice Wrapped NIBI implementation, analagous to WETH on Ethereum.
///   _   _  _____  ____  _____  _____   _    _
///  | \ | ||_   _||  _ \|_   _||  __ \ | |  | |
///  |  \| |  | |  | |_) | | |  | |__) || |  | |
///  | . ` |  | |  |  _ <  | |  |  _  / | |  | |
///  | |\  | _| |_ | |_) |_| |_ | | \ \ | |__| |
///  |_| \_||_____||____/|_____||_|  \_\ \____/
///
contract WNIBI {
    string public name = "Wrapped Nibiru";
    string public symbol = "WNIBI";
    uint8 public decimals = 18;

    event Approval(address indexed src, address indexed guy, uint wad);
    event Transfer(address indexed src, address indexed dst, uint wad);
    event Deposit(address indexed dst, uint wad);
    event Withdrawal(address indexed src, uint wad);

    mapping(address => uint) public balanceOf;
    mapping(address => mapping(address => uint)) public allowance;

    /// @dev Called automatically when the contract receives Ether with calldata,
    /// but there’s no matching function to execute.
    fallback() external payable {
        deposit();
    }

    /// @dev Called automatically when Ether is sent to the contract without any
    /// calldata (e.g., a plain transfer or send call).
    receive() external payable {
        deposit();
    }

    function deposit() public payable {
        balanceOf[msg.sender] += msg.value;
        emit Deposit(msg.sender, msg.value);
    }

    function withdraw(uint wad) public {
        require(balanceOf[msg.sender] >= wad);
        balanceOf[msg.sender] -= wad;
        payable(msg.sender).transfer(wad);
        emit Withdrawal(msg.sender, wad);
    }

    function totalSupply() public view returns (uint) {
        return address(this).balance;
    }

    function approve(address guy, uint wad) public returns (bool) {
        allowance[msg.sender][guy] = wad;
        emit Approval(msg.sender, guy, wad);
        return true;
    }

    function transfer(address dst, uint wad) public returns (bool) {
        return transferFrom(msg.sender, dst, wad);
    }

    function transferFrom(
        address src,
        address dst,
        uint wad
    ) public returns (bool) {
        require(balanceOf[src] >= wad);

        if (src != msg.sender && allowance[src][msg.sender] != type(uint).max) {
            require(allowance[src][msg.sender] >= wad);
            allowance[src][msg.sender] -= wad;
        }

        balanceOf[src] -= wad;
        balanceOf[dst] += wad;

        emit Transfer(src, dst, wad);

        return true;
    }
}
