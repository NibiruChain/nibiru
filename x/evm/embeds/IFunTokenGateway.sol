// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @dev Implements the "bankSend" functionality for sending ERC20 tokens as bank
/// coins to a Nibiru bech32 address using the "FunToken" mapping between the
/// ERC20 and bank.
interface IFunTokenGateway {
  /// @dev bankSend sends ERC20 tokens as coins to a Nibiru base account
  /// @param erc20 the address of the ERC20 token contract
  /// @param amount the amount of tokens to send
  /// @param to the receiving Nibiru base account address as a string
  function bankSend(address erc20, uint256 amount, string memory to) external;
}

address constant FUNTOKEN_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000800;

IFunTokenGateway constant FUNTOKEN_GATEWAY = IFunTokenGateway(FUNTOKEN_PRECOMPILE_ADDRESS);
