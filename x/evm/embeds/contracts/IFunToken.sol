// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @dev Implements the functionality for sending ERC20 tokens and bank
/// coins to various Nibiru accounts using either the Nibiru Bech32 address
/// using the "FunToken" mapping between the ERC20 and bank.
interface IFunToken {
    /// @dev sendToBank sends ERC20 tokens as coins to a Nibiru base account
    /// @param erc20 - the address of the ERC20 token contract
    /// @param amount - the amount of tokens to send
    /// @param to - the receiving Nibiru base account address as a string
    /// @return sentAmount - amount of tokens received by the recipient. This may
    /// not be equal to `amount` if the corresponding ERC20 contract has a fee or
    /// deduction on transfer.
    function sendToBank(
        address erc20,
        uint256 amount,
        string memory to
    ) external returns (uint256 sentAmount);
}

address constant FUNTOKEN_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000800;

IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(FUNTOKEN_PRECOMPILE_ADDRESS);
