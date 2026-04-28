// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "./IFunToken.sol";

/// @notice Malicious ERC20 used to verify VM-sender guard behavior during
/// module-originated ERC20 transfer callbacks.
contract TestERC20MaliciousCallback is ERC20 {
    constructor(
        string memory _name,
        string memory _symbol,
        uint8 // decimals_
    ) ERC20(_name, _symbol) {
        _mint(msg.sender, 1000000 * 10 ** 18);
    }

    function transfer(
        address recipient,
        uint256 amount
    ) public virtual override returns (bool) {
        // Attempt a mutable precompile sink while this transfer executes.
        // During module-originated callbacks, this must fail via VM-sender guard.
        (bool ok, bytes memory ret) = address(FUNTOKEN_PRECOMPILE).delegatecall(
            abi.encodeWithSignature(
                "bankMsgSend(string,string,uint256)",
                "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
                "unibi",
                1
            )
        );
        if (!ok && ret.length > 0) {
            assembly {
                revert(add(ret, 32), mload(ret))
            }
        }
        return super.transfer(recipient, amount);
    }
}
