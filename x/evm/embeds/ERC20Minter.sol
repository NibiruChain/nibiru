// SPDX-License-Identifier: MIT

pragma solidity 0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @dev {ERC20} token, including:
 *
 *  - ability for holders to burn (destroy) their tokens
 *  - a minter role that allows for token minting (creation)
 *
 * The contract owner is set automatically in the constructor as the
 * deployer due to "Ownable".
 * 
 * The Context contract is inherited indirectly through "ERC20" and "Ownable".
 *
 * The account that deploys the contract will be able to mint and burn tokens.
 */
contract ERC20Minter is ERC20, ERC20Burnable, Ownable {
  uint8 private _decimals;

  /**
    * @dev Grants `DEFAULT_ADMIN_ROLE`, `MINTER_ROLE` to the
    * account that deploys the contract and customizes tokens decimals
    *
    * See {ERC20-constructor}.
    */
  constructor(string memory name, string memory symbol, uint8 decimals_)
    ERC20(name, symbol) {
      _setupDecimals(decimals_);
  }

  /**
    * @dev Sets `_decimals` as `decimals_ once at Deployment'
    */
  function _setupDecimals(uint8 decimals_) private {
    _decimals = decimals_;
  }

  /**
    * @dev Overrides the `decimals()` method with custom `_decimals`
    */
  function decimals() public view virtual override returns (uint8) {
    return _decimals;
  }

  /**
    * @dev Creates `amount` new tokens for `to`.
    *
    * See {ERC20-_mint}.
    *
    * Requirements:
    *
    * - the caller must have the `MINTER_ROLE`.
    */
  function mint(address to, uint256 amount) public virtual onlyOwner {
      _mint(to, amount);
  }

   /**
   * @dev Destroys `amount` new tokens for `to`.
   *
   * See {ERC20-_burn}.
   *
   * Requirements:
   *
   * - the caller must have the `MINTER_ROLE`.
   */
  function burnCoins(address from, uint256 amount) public virtual onlyOwner {
      _burn(from, amount);
  }

}
