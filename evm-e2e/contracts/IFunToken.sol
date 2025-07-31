// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

address constant FUNTOKEN_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000800;
IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(FUNTOKEN_PRECOMPILE_ADDRESS);

import "./NibiruEvmUtils.sol";

/// @notice Implements the functionality for sending ERC20 tokens and bank
/// coins to various Nibiru accounts using either the Nibiru Bech32 address
/// using the "FunToken" mapping between the ERC20 and bank.
interface IFunToken is INibiruEvm {
    /// @notice sendToBank sends ERC20 tokens as coins to a Nibiru base account
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

    /// @notice Retrieves the ERC20 contract address associated with a given bank denomination.
    /// @param bankDenom The bank denomination string (e.g., "unibi", "erc20/0x...", "ibc/...").
    /// @return erc20Address The corresponding ERC20 contract address, or address(0) if no mapping exists.
    function getErc20Address(
        string memory bankDenom
    ) external view returns (address erc20Address);

    struct NibiruAccount {
        address ethAddr;
        string bech32Addr;
    }
    struct FunToken {
        address erc20;
        string bankDenom;
    }

    /// @notice Method "balance" returns the ERC20 balance and Bank Coin balance
    /// of some fungible token held by the given account.
    function balance(
        address who,
        address funtoken
    )
        external
        view
        returns (
            uint256 erc20Balance,
            uint256 bankBalance,
            FunToken memory token,
            NibiruAccount memory whoAddrs
        );

    /// @notice Method "bankBalance" returns the Bank Coin balance of some
    /// fungible token held by the given account.
    function bankBalance(
        address who,
        string calldata bankDenom
    )
        external
        view
        returns (uint256 bankBalance, NibiruAccount memory whoAddrs);

    /// @notice Method "whoAmI" performs address resolution for the given address
    /// string
    /// @param who Ethereum hexadecimal (EVM) address or nibi-prefixed Bech32
    /// (non-EVM) address
    /// @return whoAddrs Addresses of "who" in EVM and non-EVM formats
    function whoAmI(
        string calldata who
    ) external view returns (NibiruAccount memory whoAddrs);

    /// @notice sendToEvm transfers the caller's Bank Coins specified by `denom`
    /// to the corresponding ERC-20 representation on the EVM side. The `to`
    /// argument must be either an Ethereum hex address (0x...) or a Bech32
    /// address.
    ///
    /// The underlying logic mints (or un-escrows) the ERC-20 tokens to the `to` address if
    /// the funtoken mapping was originally minted from a coin.
    ///
    /// @param bankDenom The bank denom of the coin to send from the caller to the EVM side.
    /// @param amount The number of coins to send.
    /// @param to The Ethereum hex or bech32 address receiving the ERC-20.
    /// @return sentAmount The number of ERC-20 tokens minted or un-escrowed.
    function sendToEvm(
        string calldata bankDenom,
        uint256 amount,
        string calldata to
    ) external returns (uint256 sentAmount);

    /// @notice bankMsgSend performs a `cosmos.bank.v1beta1.MsgSend` transaction
    /// message to transfer Bank Coin funds to the given address.
    ///
    /// @param to The recipient address (hex or bech32).
    /// @param bankDenom The bank coin denom to send.
    /// @param amount The number of coins to send.
    /// @return success True if the bank send succeeded, false otherwise.
    function bankMsgSend(
        string calldata to,
        string calldata bankDenom,
        uint256 amount
    ) external returns (bool success);
}
