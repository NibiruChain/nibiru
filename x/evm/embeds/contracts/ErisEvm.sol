// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

import "./WNIBI.sol";
import "./IFunToken.sol";
import "./Wasm.sol";
import "./NibiruEvmUtils.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @title ErisEvm
///   _   _  _____  ____  _____  _____   _    _
///  | \ | ||_   _||  _ \|_   _||  __ \ | |  | |
///  |  \| |  | |  | |_) | | |  | |__) || |  | |
///  | . ` |  | |  |  _ <  | |  |  _  / | |  | |
///  | |\  | _| |_ | |_) |_| |_ | | \ \ | |__| |
///  |_| \_||_____||____/|_____||_|  \_\ \____/
///
/// @notice Delegate call interface for staking NIBI via the Eris amplifier
/// contract. This contract does not custody funds. All actions propagate
/// msg.sender via delegatecall.
contract ErisEvm {
    using Strings for uint256;

    address payable private constant WNIBI_ADDRESS =
        payable(0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97);

    /// The contract for liquid staking in Eris is the "hub". Its Rust
    /// implementation can be found here: https://github.com/erisprotocol/contracts-tokenfactory/blob/b9c993981f5190eb2fb584884471e3d8f03bd6b4/packages/eris/src/hub.rs#L147
    string private constant ERIS_WASM_CONTRACT =
        "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m";

    /// Bank denomination for stNIBI.
    string private constant ST_NIBI_DENOM =
        "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI";

    /// ERC20 address for stNIBI
    address payable private constant ST_NIBI_ADDRESS =
        payable(0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b);

    /// Bank denomination for NIBI.
    string private constant NIBI_BANK_DENOM = "unibi";

    // Define the minimum precision unit for NIBI, which is 10^12 (1e12) in 18-decimal WNIBI terms
    // This is equivalent to 1 (1e0) in 6-decimal NIBI terms.
    uint256 private constant NIBI_PRECISION_UNIT = 1e12; // 10^12

    /// @notice Query the Eris state, returning a JSON object in the form of the
    /// `LiquidStakeContract` type:
    ///
    /// ```ts
    /// export interface LiquidStakeContract {
    ///   total_ustake?: string
    ///   total_utoken?: string
    ///   exchange_rate?: string
    ///   unlocked_coins?: UnlockedCoinsEntity[]
    ///   unbonding?: string
    ///   available?: string
    ///   tvl_utoken?: string
    /// }
    /// ```
    function state() external view returns (bytes memory respJson) {
        bytes memory wasmQueryMsg = bytes('{"state":{}}');
        return WASM_PRECOMPILE.query(ERIS_WASM_CONTRACT, wasmQueryMsg);
    }

    /// @notice Deposit NIBI to liquid stake and mint stNIBI.
    function liquidStake(uint256 amount) external {
        // Require amount to be at least the NIBI_PRECISION_UNIT
        require(
            amount >= NIBI_PRECISION_UNIT,
            "Amount must be >= 1e12 wei (1 micro NIBI)"
        );
        amount = (amount / NIBI_PRECISION_UNIT) * NIBI_PRECISION_UNIT;

        string memory senderHex = Strings.toHexString(uint160(msg.sender), 20);
        IFunToken.NibiruAccount memory whoAddrs = FUNTOKEN_PRECOMPILE.whoAmI(
            senderHex
        );

        // Query Balances
        uint256 nibiWeiBalance = address(msg.sender).balance;

        // Get WNIBI balance
        uint256 wnibiBalance = IERC20(WNIBI_ADDRESS).balanceOf(msg.sender);
        // Truncate wnibiBalance to NIBI_PRECISION_UNIT (1e12)
        // This effectively zeroes out the last 12 decimals of WNIBI that are not significant for NIBI
        wnibiBalance =
            (wnibiBalance / NIBI_PRECISION_UNIT) *
            NIBI_PRECISION_UNIT;

        uint256 totalAvailable = wnibiBalance + nibiWeiBalance;
        require(totalAvailable >= amount, "Insufficient NIBI + WNIBI");

        uint256 convertWnibi = wnibiBalance >= amount ? amount : wnibiBalance;

        if (convertWnibi > 0) {
            require(
                IERC20(WNIBI_ADDRESS).transferFrom(
                    msg.sender,
                    address(this),
                    convertWnibi
                ),
                "WNIBI transferFrom failed"
            );
            WNIBI(WNIBI_ADDRESS).withdraw(convertWnibi);

            // ErisEvm transfers the unwrapped NIBI back to msg.sender's EVM address.
            // Due to Nibiru's architecture, this will correctly populate the user's
            // wei balance for the WASM call.
            payable(msg.sender).transfer(convertWnibi);
        }

        bytes memory wasmMsg = bytes('{"bond":{}}');

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: NIBI_BANK_DENOM,
            amount: amount / NIBI_PRECISION_UNIT
        });

        (uint256 beforeBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
            whoAddrs.ethAddr,
            ST_NIBI_DENOM
        );

        _doWasmExecute(ERIS_WASM_CONTRACT, wasmMsg, funds);

        (uint256 afterBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
            whoAddrs.ethAddr,
            ST_NIBI_DENOM
        );
        // It's crucial that stNIBI is minted for the staking operation to be considered successful.
        // If afterBalance is NOT greater than beforeBalance, it means the bond operation
        // either failed to mint stNIBI or minted zero, and the transaction should revert.
        require(
            afterBalance > beforeBalance,
            "Wasm operation failed to mint stNIBI or minted zero."
        );
        uint256 delta = afterBalance - beforeBalance;
        _doSendToEvm(ST_NIBI_DENOM, delta, senderHex);
    }

    /// @notice Redeem any stNIBI that has finished unstaking to receive the NIBI
    /// principal and any accrued rewards from liquid staking. NIBI received is
    /// converted to WNIBI.
    function redeem() external {
        uint256 nibiBalBefore = address(msg.sender).balance;

        bytes memory wasmMsg = bytes('{"withdraw_unbonded":{}}');
        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](0);
        _doWasmExecute(ERIS_WASM_CONTRACT, wasmMsg, funds);

        uint256 nibiBalAfter = address(msg.sender).balance;
        uint256 nibiReceived = nibiBalAfter > nibiBalBefore
            ? nibiBalAfter - nibiBalBefore
            : 0;
        if (nibiReceived > 0) {
            WNIBI(WNIBI_ADDRESS).deposit{value: nibiBalAfter - nibiBalBefore}();
        }
    }

    /// @notice Queue to unstake stNIBI to later redeem it for the principal and
    /// accrued rewards from liquid staking.
    function unstake(uint256 stAmount) external {
        // See the other address format of the sender
        string memory senderHex = Strings.toHexString(uint160(msg.sender), 20);

        (
            uint256 erc20Balance,
            uint256 bankBalance,
            IFunToken.FunToken memory token,

        ) = FUNTOKEN_PRECOMPILE.balance(msg.sender, ST_NIBI_ADDRESS);

        uint256 totalAvailable = erc20Balance + bankBalance;
        require(totalAvailable >= stAmount, "Insufficient stNIBI balance");

        uint256 convertErc20 = bankBalance >= stAmount
            ? 0
            : stAmount - bankBalance;
        if (convertErc20 > 0) {
            // Convert some of the ERC20 balance over to Bank for use with Wasm.
            _doSendToBank(token.erc20, convertErc20, senderHex);
        }

        bytes memory wasmMsg = bytes('{"queue_unbond":{}}');

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: ST_NIBI_DENOM,
            amount: stAmount
        });

        _doWasmExecute(ERIS_WASM_CONTRACT, wasmMsg, funds);
    }

    function _doSendToBank(
        address token,
        uint256 amount,
        string memory recipient
    ) internal {
        SafeDelegateCall.call(
            FUNTOKEN_PRECOMPILE_ADDRESS,
            abi.encodeWithSignature(
                "sendToBank(address,uint256,string)",
                token,
                amount,
                recipient
            ),
            "sendToBank failed"
        );
    }

    function _doWasmExecute(
        string memory contractAddress,
        bytes memory msgExecute,
        INibiruEvm.BankCoin[] memory funds
    ) internal returns (bytes memory) {
        return
            SafeDelegateCall.call(
                WASM_PRECOMPILE_ADDRESS,
                abi.encodeWithSelector(
                    IWasm.execute.selector,
                    contractAddress,
                    msgExecute,
                    funds
                ),
                "Wasm execute failed"
            );
    }

    function _doSendToEvm(
        string memory denom,
        uint256 amount,
        string memory recipient
    ) internal {
        SafeDelegateCall.call(
            FUNTOKEN_PRECOMPILE_ADDRESS,
            abi.encodeWithSignature(
                "sendToEvm(string,uint256,string)",
                denom,
                amount,
                recipient
            ),
            "sendToEvm failed"
        );
    }
}

library SafeDelegateCall {
    /// @dev Performs a delegatecall and:
    ///  - returns the raw bytes on success
    ///  - re-throws the original revert reason if one exists
    ///  - otherwise reverts with `fallbackMsg`
    function call(
        address target,
        bytes memory data,
        string memory fallbackMsg
    ) internal returns (bytes memory result) {
        (bool success, bytes memory returndata) = target.delegatecall(data);

        if (success) {
            return returndata;
        }

        // If the callee included a revert reason, bubble it up verbatim.
        if (returndata.length > 0) {
            /// @solidity memory-safe-assembly
            assembly {
                let size := mload(returndata)
                revert(add(returndata, 32), size)
            }
        }

        // Otherwise use the provided fallback message.
        revert(fallbackMsg);
    }
}
