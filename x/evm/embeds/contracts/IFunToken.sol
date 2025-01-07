// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

address constant FUNTOKEN_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000800;
IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(FUNTOKEN_PRECOMPILE_ADDRESS);

import "./NibiruEvmUtils.sol";

/// @dev Implements the functionality for sending ERC20 tokens and bank
/// coins to various Nibiru accounts using either the Nibiru Bech32 address
/// using the "FunToken" mapping between the ERC20 and bank.
interface IFunToken is INibiruEvm {
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

    /**
     * @dev sendToEvm transfers the caller's bank coin `denom` to its ERC-20 representation on the EVM side.
     * The `to` argument must be either an Ethereum hex address (0x...) or a Bech32 address.
     *
     * The underlying logic mints (or un-escrows) the ERC-20 tokens to the `to` address if
     * the funtoken mapping was originally minted from a coin.
     *
     * @param bankDenom The bank denom of the coin to send from the caller to the EVM side.
     * @param amount The number of coins to send.
     * @param to The Ethereum hex or bech32 address receiving the ERC-20.
     * @return sentAmount The number of ERC-20 tokens minted or un-escrowed.
     */
    function sendToEvm(
        string calldata bankDenom,
        uint256 amount,
        string calldata to
    ) external returns (uint256 sentAmount);

    /**
     * @dev bankMsgSend performs a `cosmos.bank.v1beta1.MsgSend` from the caller
     * into the Cosmos side, akin to running the standard `bank` module's send operation.
     *
     * @param to The recipient address (hex or bech32).
     * @param bankDenom The bank coin denom to send.
     * @param amount The number of coins to send.
     * @return success True if the bank send succeeded, false otherwise.
     */
    function bankMsgSend(
        string calldata to,
        string calldata bankDenom,
        uint256 amount
    ) external returns (bool success);
}
