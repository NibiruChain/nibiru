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

    address payable private constant WNIBI_ADDRESS = payable(
        0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97
    );

    /// The contract for liquid staking in Eris is the "hub". Its Rust
    /// implementation can be found here: https://github.com/erisprotocol/contracts-tokenfactory/blob/b9c993981f5190eb2fb584884471e3d8f03bd6b4/packages/eris/src/hub.rs#L147
    string private constant ERIS_WASM_CONTRACT =
        "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m";

    /// @dev Escrow and gauge are separate Wasm contracts. Their Rust message
    /// schemas are packages/eris/src/voting_escrow.rs and amp_gauges.rs at
    /// https://github.com/erisprotocol/contracts-tokenfactory/tree/7c3acdb32907d30727bd8f4313bd91a539aaa8d8.
    string private constant ERIS_ESCROW =
        "nibi1us4rh4a9rexvde8l3m8f8nlz6wcf9qg57zk2w06kqkm4vtp96vkst9kuwd";
    string private constant ERIS_GAUGE =
        "nibi1qh59hdelxfwah7g7e8k0lxu4upqatq5w4jdw60yv9shhlhm86ckq4txl92";

    /// Bank denomination for stNIBI.
    string private constant ST_NIBI_DENOM =
        "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI";

    /// ERC20 address for stNIBI
    address payable private constant ST_NIBI_ADDRESS = payable(
        0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b
    );

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
            (wnibiBalance / NIBI_PRECISION_UNIT) * NIBI_PRECISION_UNIT;

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
    /// principal and any accrued rewards from liquid staking.
    function redeem() external {
        bytes memory wasmMsg = bytes('{"withdraw_unbonded":{}}');
        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](0);
        _doWasmExecute(ERIS_WASM_CONTRACT, wasmMsg, funds);
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

        uint256 convertErc20 =
            bankBalance >= stAmount ? 0 : stAmount - bankBalance;
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

    /// @notice Identifies support for escrow and allocation voting on an upgraded proxy.
    /// @dev Constants add no proxy storage slots. Clients must check this before signing.
    /// @return Supported governance interface version, currently 1.
    function governanceVersion() external pure returns (uint256) {
        return 1;
    }

    /// @notice Lock stNIBI receipts to obtain allocation voting power.
    /// @param amount Receipt amount in six-decimal base units, identical for Bank and ERC20.
    /// @param time Lock duration in seconds; Eris rounds down to whole weeks.
    function createLock(uint256 amount, uint64 time) external {
        _depositEscrow(
            amount,
            abi.encodePacked(
                '{"create_lock":{"time":',
                uint256(time).toString(),
                "}}"
            )
        );
    }

    /// @notice Add receipts to an existing active escrow without changing its expiry.
    /// @param amount Receipt amount in six-decimal base units.
    function increaseLockAmount(uint256 amount) external {
        _depositEscrow(
            amount,
            bytes('{"extend_lock_amount":{"extend_to_min_periods":false}}')
        );
    }

    /// @notice Extend an active lock, or relock an expired position from the current period.
    /// @param time Duration to add in seconds. The Rust contract enforces remaining-time bounds.
    function extendLockTime(uint64 time) external {
        _extendLock(time);
    }

    /// @notice Withdraw an expired escrow and return its receipts as ERC20 stNIBI.
    /// @dev Converts only the Bank balance increase from this withdrawal. Existing Bank receipts stay put.
    function withdrawEscrow() external {
        (uint256 beforeBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
            msg.sender,
            ST_NIBI_DENOM
        );
        _doWasmExecute(
            ERIS_ESCROW,
            bytes('{"withdraw":{}}'),
            new INibiruEvm.BankCoin[](0)
        );
        (uint256 afterBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
            msg.sender,
            ST_NIBI_DENOM
        );
        uint256 returned = afterBalance - beforeBalance;
        if (returned > 0)
            _doSendToEvm(
                ST_NIBI_DENOM,
                returned,
                Strings.toHexString(uint160(msg.sender), 20)
            );
    }

    /// @notice Replace the caller's allocation preferences for the next weekly period.
    /// @param validators Eligible Nibiru validator operator addresses.
    /// @param weights Basis points summing to 10000. Every validator must be unique.
    function vote(
        string[] calldata validators,
        uint16[] calldata weights
    ) external {
        _vote(validators, weights);
    }

    /// @notice Relock or extend and replace allocation preferences atomically.
    /// @dev Escrow callbacks first refresh existing preferences. A subsequent vote failure
    /// reverts those callbacks and the extension together through EVM transaction rollback.
    /// @param time Duration to add in seconds.
    /// @param validators Eligible Nibiru validator operator addresses.
    /// @param weights Basis points summing to 10000.
    function extendLockAndVote(
        uint64 time,
        string[] calldata validators,
        uint16[] calldata weights
    ) external {
        // Validate before executing callbacks or changing the escrow.
        bytes memory args = _voteArgs(validators, weights);
        _extendLock(time);
        _doWasmExecute(ERIS_GAUGE, args, new INibiruEvm.BankCoin[](0));
    }

    function _extendLock(uint64 time) internal {
        _doWasmExecute(
            ERIS_ESCROW,
            abi.encodePacked(
                '{"extend_lock_time":{"time":',
                uint256(time).toString(),
                "}}"
            ),
            new INibiruEvm.BankCoin[](0)
        );
    }

    /// @dev Bank receipts fund the deposit first. FunToken transfers the delegated
    /// caller's ERC20 shortfall directly, so no allowance or proxy custody is needed.
    function _depositEscrow(uint256 amount, bytes memory args) internal {
        require(amount > 0, "Zero receipt amount");
        (uint256 erc20Balance, uint256 bankBalance, , ) = FUNTOKEN_PRECOMPILE
            .balance(msg.sender, ST_NIBI_ADDRESS);
        require(
            bankBalance >= amount || erc20Balance >= amount - bankBalance,
            "Insufficient stNIBI"
        );
        if (bankBalance < amount)
            _doSendToBank(
                ST_NIBI_ADDRESS,
                amount - bankBalance,
                Strings.toHexString(uint160(msg.sender), 20)
            );
        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({denom: ST_NIBI_DENOM, amount: amount});
        _doWasmExecute(ERIS_ESCROW, args, funds);
    }

    function _vote(
        string[] calldata validators,
        uint16[] calldata weights
    ) internal {
        _doWasmExecute(
            ERIS_GAUGE,
            _voteArgs(validators, weights),
            new INibiruEvm.BankCoin[](0)
        );
    }

    /// @dev Restrict strings to the canonical address shape before JSON interpolation.
    /// Rust's allowlist check verifies the address itself. No quotes or JSON delimiters
    /// can pass this character filter; do not replace it with a prefix-only check.
    function _voteArgs(
        string[] calldata validators,
        uint16[] calldata weights
    ) internal pure returns (bytes memory args) {
        require(
            validators.length == weights.length &&
                validators.length > 0 &&
                validators.length <= 30,
            "Invalid vote arrays"
        );
        args = bytes('{"vote":{"votes":[');
        uint256 total;
        bytes memory alphabet = bytes("023456789acdefghjklmnpqrstuvwxyz");
        for (uint256 i; i < validators.length; ++i) {
            bytes memory addr = bytes(validators[i]);
            require(addr.length == 50, "Invalid validator length");
            bytes memory prefix = bytes("nibivaloper1");
            for (uint256 j; j < 12; ++j)
                require(addr[j] == prefix[j], "Invalid validator prefix");
            for (uint256 j = 12; j < addr.length; ++j) {
                bool valid;
                for (uint256 k; k < alphabet.length; ++k)
                    if (addr[j] == alphabet[k]) {
                        valid = true;
                        break;
                    }
                require(valid, "Invalid validator character");
            }
            for (uint256 j; j < i; ++j)
                require(
                    keccak256(bytes(validators[j])) != keccak256(addr),
                    "Duplicate validator"
                );
            require(weights[i] > 0, "Zero vote weight");
            total += weights[i];
            args = abi.encodePacked(
                args,
                i == 0 ? "" : ",",
                '["',
                addr,
                '",',
                uint256(weights[i]).toString(),
                "]"
            );
        }
        require(total == 10000, "Vote must total 10000");
        return abi.encodePacked(args, "]}}");
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
