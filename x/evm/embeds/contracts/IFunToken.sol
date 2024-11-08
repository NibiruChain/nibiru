// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

address constant FUNTOKEN_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000800;
IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(FUNTOKEN_PRECOMPILE_ADDRESS);

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
        string calldata to
    ) external returns (uint256 sentAmount);

    struct NibiruAccount {
        address ethAddr;
        string bech32Addr;
    }
    struct FunToken {
        address erc20;
        string bankDenom;
    }

    function balance(
        address who,
        address funtoken
    )
        external
        returns (
            uint256 erc20Balance,
            uint256 bankBalance,
            FunToken memory token,
            NibiruAccount memory whoAddrs
        );

    function bankBalance(
        address who,
        string calldata bankDenom
    ) external returns (uint256 bankBalance, NibiruAccount memory whoAddrs);

    function whoAmI(
        string calldata who
    ) external returns (NibiruAccount memory whoAddrs);
}
