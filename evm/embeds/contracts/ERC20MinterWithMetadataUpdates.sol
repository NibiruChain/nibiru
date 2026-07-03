// SPDX-License-Identifier: MIT

pragma solidity >=0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/// @dev {ERC20} token, including:
///
///  - an "owner" that can mint tokens
///  - ability for holders to burn (destroy) their tokens
///
/// The contract owner is set automatically in the constructor as the
/// deployer due to "Ownable".
///
/// The Context contract is inherited indirectly through "ERC20" and "Ownable".
contract ERC20MinterWithMetadataUpdates is ERC20, ERC20Burnable, Ownable {
    uint8 private _decimals;

    // use our own state variables instead of the ones from ERC20 to allow for updating
    string private _name;
    string private _symbol;

    /// @dev Grants "owner" status to the account that deploys the contract and
    /// customizes tokens decimals.
    ///
    /// See {ERC20-constructor}.
    constructor(
        string memory name_,
        string memory symbol_,
        uint8 decimals_
    ) ERC20(name_, symbol_) {
        _decimals = decimals_;
        _name = name_;
        _symbol = symbol_;
    }

    /// @dev Overrides the `decimals()` method with custom `_decimals`
    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }

    /// @dev Overrides the `name()` method to return the current name of the token.
    function name() public view virtual override returns (string memory) {
        // This function returns the name of the token.
        // It's a simple override to provide the current name of the token.
        return _name;
    }

    /// @dev Overrides the `symbol()` method to return the current symbol of the token.
    function symbol() public view virtual override returns (string memory) {
        // This function returns the symbol of the token.
        // It's a simple override to provide the current symbol of the token.
        return _symbol;
    }

    /// @dev Allows the owner to update the decimals of the token.
    ///
    /// This function can be used to change the decimals of the token after
    /// deployment. However, changing the decimals of a token can lead to
    /// complications, especially with existing balances and should be done
    /// with caution. It's generally recommended to set the decimals once at
    /// deployment and not change it unless absolutely necessary.
    function setDecimals(uint8 decimals_) public onlyOwner {
        // This function allows the owner to update the decimals of the token.
        // Note: This is a simple example and in practice, changing decimals
        // after deployment can lead to complications and should be done with caution.
        _decimals = decimals_;
    }

    /// @dev Allows the owner to update the name of the token.
    ///
    /// This function can be used to change the name of the token after
    /// deployment. However, changing the name of a token can lead to
    /// complications, especially with existing balances and should be done
    /// with caution. It's generally recommended to set the name once at
    /// deployment and not change it unless absolutely necessary.
    function setName(string memory name_) public onlyOwner {
        // This function allows the owner to update the name of the token.
        // Note: This is a simple example and in practice, changing the name
        // after deployment may not be common.
        _name = name_;
    }

    /// @dev Allows the owner to update the symbol of the token.
    ///
    /// This function can be used to change the symbol of the token after
    /// deployment. However, changing the symbol of a token can lead to
    /// complications, especially with existing balances and should be done
    /// with caution. It's generally recommended to set the symbol once at
    /// deployment and not change it unless absolutely necessary.
    function setSymbol(string memory symbol_) public onlyOwner {
        // This function allows the owner to update the symbol of the token.
        // Note: This is a simple example and in practice, changing the symbol
        // after deployment may not be common.
        _symbol = symbol_;
    }

    /// @dev Creates `amount` new tokens for `to`.
    ///
    /// See {ERC20-_mint}.
    function mint(address to, uint256 amount) public virtual onlyOwner {
        _mint(to, amount);
    }

    /// @dev Destroys `amount` new tokens for `to`. Suitable when the contract owner
    /// should have authority to burn tokens from an account directly, such as in
    /// the case of regulatory compliance, or actions selected via
    /// decentralized governance.
    ///
    /// See {ERC20-_burn}.
    function burnFromAuthority(
        address from,
        uint256 amount
    ) public virtual onlyOwner {
        _burn(from, amount);
    }
}
