// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.8.19;

import "@nibiruchain/solidity/contracts/IFunToken.sol";
import "@nibiruchain/solidity/contracts/Wasm.sol";
import "@nibiruchain/solidity/contracts/NibiruEvmUtils.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
// import "@openzeppelin/contracts/utils/Strings.sol"; // Already imported

library SafeDelegateCall {
    /**
     * @dev Performs a delegatecall and:
     *      • returns the raw bytes on success
     *      • re-throws the original revert reason if one exists
     *      • otherwise reverts with `fallbackMsg`
     */
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

contract PerpVaultEvmInterface is Ownable {
    using Strings for uint256; // For Strings.toString(uint256)

    string public perpContractAddress;
    // address public owner; // Replaced by Ownable

    uint256 public timelockDelay; // Delay in seconds for address change
    string public pendingPerpContractAddress;
    uint256 public changeEffectiveTimestamp;

    uint256 public constant PRECISION = 1e18;

    event PerpContractAddressChangeProposed(
        string indexed newAddress,
        uint256 effectiveTimestamp
    );
    event PerpContractAddressChanged(
        string indexed oldAddress,
        string indexed newAddress
    );

    constructor(
        string memory _initialPerpContractAddress,
        uint256 _timelockDelay
    ) {
        require(
            bytes(_initialPerpContractAddress).length > 0,
            "Initial Perp address empty"
        );
        perpContractAddress = _initialPerpContractAddress;
        timelockDelay = _timelockDelay;
        _transferOwnership(msg.sender);
    }

    /**
     * @notice Proposes a new Perp contract address.
     * @dev Can only be called by the owner. Starts the timelock period.
     * @param _newAddress The proposed new Bech32 address for the Perp contract.
     */
    function proposeNewPerpContractAddress(
        string memory _newAddress
    ) external onlyOwner {
        require(bytes(_newAddress).length > 0, "New Perp address empty");
        require(
            keccak256(bytes(_newAddress)) !=
                keccak256(bytes(perpContractAddress)),
            "New address is same as current"
        );
        pendingPerpContractAddress = _newAddress;
        changeEffectiveTimestamp = block.timestamp + timelockDelay;
        emit PerpContractAddressChangeProposed(
            _newAddress,
            changeEffectiveTimestamp
        );
    }

    /**
     * @notice Executes the proposed change to the Perp contract address.
     * @dev Can only be called by the owner after the timelock period has passed.
     */
    function executeNewPerpContractAddress() external onlyOwner {
        require(
            bytes(pendingPerpContractAddress).length > 0,
            "No pending change"
        );
        require(
            block.timestamp >= changeEffectiveTimestamp,
            "Timelock not expired"
        );

        string memory oldAddress = perpContractAddress;
        perpContractAddress = pendingPerpContractAddress;

        // Clear pending change
        pendingPerpContractAddress = "";
        changeEffectiveTimestamp = 0;

        emit PerpContractAddressChanged(oldAddress, perpContractAddress);
    }

    event DepositCollateral(
        address indexed sender,
        uint256 erc20Amount,
        string vaultAddress,
        string bankDenom,
        address bankAddress
    );

    function _performSendToBank(
        address token,
        uint256 amount,
        string memory recipient
    ) public returns (uint256) {
        bytes memory ret = SafeDelegateCall.call(
            FUNTOKEN_PRECOMPILE_ADDRESS,
            abi.encodeWithSignature(
                "sendToBank(address,uint256,string)",
                token,
                amount,
                recipient
            ),
            "sendToBank failed"
        );
        return abi.decode(ret, (uint256));
    }

    function _performWasmExecute(
        string memory contractAddress,
        bytes memory msgExecute,
        INibiruEvm.BankCoin[] memory funds
    ) public returns (bytes memory result) {
        return
            SafeDelegateCall.call(
                WASM_PRECOMPILE_ADDRESS,
                abi.encodeWithSelector(
                    IWasm.execute.selector,
                    contractAddress,
                    msgExecute,
                    funds
                ),
                "Wasm execute failed test"
            );
    }

    function _performSendToEvm(
        string memory denom,
        uint256 amount,
        string memory recipient
    ) public {
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

    /* ----------------------------------------------------- */
    /* -------------------     VAULT     ------------------- */
    /* ----------------------------------------------------- */
    /**
     * @notice Deposits collateral into a specified vault.
     * @dev This function facilitates depositing collateral, which can be sourced from native bank tokens
     *      or ERC20 tokens. It interacts with a Wasm-based vault. If specified, it can also
     *      transfer the resulting vault shares from the user's bank account to their EVM address.
     *
     *      The `wasmMsgExecute` parameter is crucial as it contains the specific Wasm message
     *      (e.g., `{"deposit":{"amount":"<depositAmount>"}}`) required by the target vault contract
     *      to process the deposit. The `depositAmount` parameter of this function is used to specify
     *      the funds sent alongside the Wasm message. If the Wasm message itself contains an amount field,
     *      it should be consistent with this `depositAmount`.
     *
     *      Workflow:
     *      1. Validates that `depositAmount` is greater than zero.
     *      2. If `sendSharesToEvm` is true or `useErc20Amount > 0`, retrieves the sender's associated Nibiru addresses.
     *      3. If `sendSharesToEvm` is true:
     *         a. Queries the vault (`_vaultAddress`) for its shares denomination.
     *         b. Records the sender's current balance of these shares in their bank account.
     *      4. Retrieves the bank denomination corresponding to the `collateralAddress` (ERC20 token).
     *      5. If `useErc20Amount > 0`, transfers this amount of ERC20 tokens from `msg.sender` to their bank account,
     *         converting them to their bank denomination.
     *      6. Prepares the funds (consisting of `depositAmount` of the collateral's bank_denom) to be sent with the Wasm execution.
     *      7. Executes the Wasm message (`wasmMsgExecute`) on the `_vaultAddress` with the prepared funds. This performs the actual deposit into the Wasm vault.
     *      8. If `sendSharesToEvm` is true:
     *         a. Retrieves the sender's balance of vault shares in their bank account after the deposit.
     *         b. If the balance has increased, calculates the difference (newly minted shares).
     *         c. Transfers these new shares from the sender's bank account to their EVM address (`msg.sender`).
     *      9. Emits a `DepositCollateral` event.
     *
     * @param wasmMsgExecute The Wasm execute message (as bytes) to be sent to the vault contract for processing the deposit.
     *                       This message must conform to the target vault's API for deposits.
     *                       Example: `bytes('{"deposit":{"collateral":{"amount":"1000000","denom":"unibi"}}}')` or `bytes('{"deposit":{}}')` if amount is passed via funds.
     * @param depositAmount The total amount of collateral (denominated in the collateral's native bank token, e.g., uNUSD or uATOM) to be deposited into the vault.
     *                      This amount is sent as `funds` with the `wasmMsgExecute` call.
     * @param useErc20Amount The amount of ERC20 tokens (from `collateralAddress`) to be used for the deposit.
     *                       If this value is greater than 0, these ERC20 tokens are first transferred from the sender's
     *                       EVM address to their bank account. The `depositAmount` should reflect the total intended deposit,
     *                       inclusive of any amount sourced from ERC20s.
     * @param _vaultAddress The string representation of the Wasm vault contract address (e.g., a Bech32 address).
     * @param collateralAddress The EVM address of the ERC20 token to be used as collateral.
     * @param sendSharesToEvm A boolean flag. If true, any vault shares minted as a result of the deposit
     *                        will be transferred from the user's bank account to their EVM address (`msg.sender`).
     *                        If false, shares remain in the user's bank account.
     */
    function deposit(
        bytes memory wasmMsgExecute,
        uint256 depositAmount,
        uint256 useErc20Amount,
        string memory _vaultAddress,
        address collateralAddress,
        bool sendSharesToEvm
    ) external {
        require(depositAmount > 0, "Deposit amount must be greater than zero");

        string memory sharesDenom;
        uint256 beforeBalance = 0;
        IFunToken.NibiruAccount memory whoAddrs;

        if (sendSharesToEvm || useErc20Amount > 0) {
            whoAddrs = FUNTOKEN_PRECOMPILE.whoAmI(
                Strings.toHexString(uint160(msg.sender), 20)
            );
        }

        if (sendSharesToEvm) {
            sharesDenom = removeQuotes(
                string(
                    WASM_PRECOMPILE.query(
                        _vaultAddress,
                        bytes('{"get_vault_share_denom":{}}')
                    )
                )
            );

            (beforeBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
                whoAddrs.ethAddr,
                sharesDenom
            );
        }

        (, , IFunToken.FunToken memory token, ) = FUNTOKEN_PRECOMPILE.balance(
            msg.sender,
            collateralAddress
        );

        if (useErc20Amount > 0) {
            _performSendToBank(
                collateralAddress,
                useErc20Amount,
                whoAddrs.bech32Addr
            );
        }

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: token.bankDenom,
            amount: depositAmount
        });

        _performWasmExecute(_vaultAddress, wasmMsgExecute, funds);

        if (sendSharesToEvm) {
            (uint256 afterBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
                whoAddrs.ethAddr,
                sharesDenom
            );

            if (afterBalance > beforeBalance) {
                uint256 delta = afterBalance - beforeBalance;

                _performSendToEvm(
                    sharesDenom,
                    delta,
                    Strings.toHexString(uint160(msg.sender), 20)
                );
            }
        }

        emit DepositCollateral(
            msg.sender,
            useErc20Amount,
            _vaultAddress,
            token.bankDenom,
            collateralAddress
        );
    }

    /**
     * @notice Redeem (withdraw) vault shares for the underlying collateral.
     * @param _vaultAddress   Bech32 Wasm vault address.
     * @param wasmMsgExecute  JSON-encoded execute msg for the vault’s `redeem` entry-point.
     * @param redeemAmount    Amount of vault shares (in BANK units) to redeem.
     * @param sendCollateralToEvm  If true, any newly-received collateral is bridged back to the EVM.
     */
    function redeem(
        bytes memory wasmMsgExecute,
        string memory _vaultAddress,
        uint256 redeemAmount,
        bool sendCollateralToEvm
    ) external {
        require(redeemAmount > 0, "Redeem amount must be > 0");

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](0);
        _performWasmExecute(_vaultAddress, wasmMsgExecute, funds);

        if (sendCollateralToEvm) {
            string memory collateralDenom = removeQuotes(
                string(
                    WASM_PRECOMPILE.query(
                        _vaultAddress,
                        bytes('{"get_collateral_denom":{}}')
                    )
                )
            );
            uint256 beforeBalance = 0;
            IFunToken.NibiruAccount memory whoAddrs;
            whoAddrs = FUNTOKEN_PRECOMPILE.whoAmI(
                Strings.toHexString(uint160(msg.sender), 20)
            );
            (beforeBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
                whoAddrs.ethAddr,
                collateralDenom
            );
            (uint256 afterBalance, ) = FUNTOKEN_PRECOMPILE.bankBalance(
                whoAddrs.ethAddr,
                collateralDenom
            );

            if (afterBalance > beforeBalance) {
                uint256 delta = afterBalance - beforeBalance;
                _performSendToEvm(
                    collateralDenom,
                    delta,
                    Strings.toHexString(uint160(msg.sender), 20)
                );
            }
        }
    }

    /**
     * @notice Submit a withdraw request by sending shares to the vault (timelocked unlock).
     *         If the caller holds share ERC-20s on the EVM side, they can first convert
     *         them back into BANK tokens via `useErc20Amount`.
     *
     * @param _vaultAddress     Bech32 Wasm vault address.
     * @param wasmMsgExecute    JSON-encoded execute msg for the vault’s `make_withdraw_request` entry-point.
     * @param withdrawAmount    Amount of shares (BANK units) to withdraw/lock.
     * @param useErc20Amount    If > 0, number of share ERC-20 tokens to first bridge into BANK.
     */
    function makeWithdrawRequest(
        string memory _vaultAddress,
        bytes memory wasmMsgExecute,
        uint256 withdrawAmount,
        uint256 useErc20Amount
    ) external {
        require(withdrawAmount > 0, "Withdraw amount must be > 0");

        string memory sharesDenom = removeQuotes(
            string(
                WASM_PRECOMPILE.query(
                    _vaultAddress,
                    bytes('{"get_vault_share_denom":{}}')
                )
            )
        );

        IFunToken.NibiruAccount memory whoAddrs;
        address shareErc20 = FUNTOKEN_PRECOMPILE.getErc20Address(sharesDenom);

        if (useErc20Amount > 0) {
            whoAddrs = FUNTOKEN_PRECOMPILE.whoAmI(
                Strings.toHexString(uint160(msg.sender), 20)
            );

            _performSendToBank(shareErc20, useErc20Amount, whoAddrs.bech32Addr);
        }

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: sharesDenom,
            amount: withdrawAmount
        });

        _performWasmExecute(_vaultAddress, wasmMsgExecute, funds);
    }

    /**
     * @notice Gets the ERC20 address corresponding to the vault's collateral denom.
     * @param _vaultAddress The Bech32 address of the vault Wasm contract.
     * @return collateralAddress The ERC20 address of the collateral token.
     */
    function getCollateralErc20AddressOfVault(
        string memory _vaultAddress
    ) public view returns (address collateralAddress) {
        string memory bankDenom = string(
            WASM_PRECOMPILE.query(
                _vaultAddress,
                bytes(string(abi.encodePacked('{"get_collateral_denom":{}}')))
            )
        );

        if (bytes(bankDenom).length == 0) {
            revert("Bank denom is empty");
        }
        bankDenom = removeQuotes(bankDenom);

        collateralAddress = FUNTOKEN_PRECOMPILE.getErc20Address(bankDenom);
        return collateralAddress;
    }

    /* ----------------------------------------------------- */
    /* -------------------     PERP      ------------------- */
    /* ----------------------------------------------------- */
    /**
     * @notice Opens a new trade on the perpetuals market.
     * @dev This function allows a user to open a trade by specifying the Wasm message for the
     *      perpetuals contract, the collateral to use, the trade amount, and an optional amount
     *      of ERC20 tokens to first transfer to their bank balance.
     *      It first determines the collateral address based on `collateralIndex`.
     *      If `useERC20Amount` is greater than zero, it transfers that amount of the specified
     *      ERC20 collateral from `msg.sender` to their corresponding bank module address.
     *      It then checks if the `msg.sender` has sufficient balance of the collateral in the bank module
     *      to cover the `tradeAmount`.
     *      Finally, it constructs the funds array and calls `_performWasmExecute` to send the
     *      `wasmMsgExecute` message along with the `tradeAmount` as funds to the `perpContractAddress`.
     *
     *      IMPORTANT: The wasmMsgExecute should include "is_evm_origin": true to ensure that
     *      funds are properly returned to the EVM address when the trade is closed.
     *
     * @param wasmMsgExecute The Wasm execute message (e.g., for opening a position) to be sent to the perpetuals contract.
     *                       Must include "is_evm_origin": true for proper fund handling.
     * @param collateralIndex The index used to identify the collateral token.
     * @param tradeAmount The amount of collateral (in its native bank denomination) to be used for the trade.
     *                    This amount is taken from the user's bank balance.
     * @param useERC20Amount The amount of ERC20 collateral to transfer from the user's ERC20 balance
     *                       to their bank module balance before opening the trade. If 0, no ERC20 transfer occurs.
     *                       This is useful if the user holds collateral as ERC20s and needs to move it to the
     *                       bank module to be used by the perpetuals contract.
     * @dev Requirements:
     *      - `tradeAmount` must be greater than 0.
     *      - The `msg.sender` must have a bank balance of the specified collateral greater than or equal to `tradeAmount`
     *        (after any `useERC20Amount` transfer).
     */
    function openTrade(
        bytes memory wasmMsgExecute,
        uint256 collateralIndex, // to avoid unpacking from wasmMsgExecute
        uint256 tradeAmount,
        uint256 useERC20Amount
    ) external {
        address collateralAddress = FUNTOKEN_PRECOMPILE.getErc20Address(
            getDenomOfCollateralIndex(collateralIndex)
        );

        require(tradeAmount > 0, "Collateral amount must be > 0");

        if (useERC20Amount > 0) {
            //who query
            IFunToken.NibiruAccount memory whoAddrs = FUNTOKEN_PRECOMPILE
                .whoAmI(Strings.toHexString(uint160(msg.sender), 20));

            _performSendToBank(
                collateralAddress,
                useERC20Amount, // Use the separate parameter
                whoAddrs.bech32Addr
            );
        }

        (
            ,
            uint256 bankBalance,
            IFunToken.FunToken memory token,

        ) = FUNTOKEN_PRECOMPILE.balance(msg.sender, collateralAddress);

        require(
            bankBalance >= tradeAmount,
            "Trade amount exceeds available collateral"
        );

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: token.bankDenom,
            amount: tradeAmount
        });

        _performWasmExecute(perpContractAddress, wasmMsgExecute, funds);
    }

    /**
     * @notice Closes a market trade on the Perp AMM module.
     * @dev This function constructs and executes a Wasm message to the `perpContractAddress`
     *      to close a specified trade. The fund handling is now managed by the perp contract
     *      based on the trade's is_evm_origin flag, which was set when the trade was opened.
     *
     *      For trades opened via this EVM interface (is_evm_origin = true), the perp contract
     *      automatically converts funds to ERC20 and sends them to the trader's EVM address.
     *      For trades opened directly on Cosmos (is_evm_origin = false), funds remain
     *      in the trader's bank balance.
     *
     * @param tradeId The unique identifier of the trade to be closed.
     * @return result The raw byte string returned by the Wasm execution of the `close_trade`
     *                message. This typically contains data about the outcome of the operation,
     *                such as details of the closed trade or error information.
     */
    function closeTradeMarket(uint256 tradeId) external returns (bytes memory) {
        bytes memory wasmMsgExecute = abi.encodePacked(
            '{"close_trade":{"trade_index":"UserTradeIndex(',
            tradeId.toString(), // Use .toString() from OpenZeppelin's Strings for uint256
            ')"}}'
        );

        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](0); // No funds sent for close_trade

        bytes memory result = _performWasmExecute(
            perpContractAddress,
            wasmMsgExecute,
            funds
        );

        return result;
    }

    /**
     * @notice Executes Perp Wasm messages that do not involve fund transfers.
     * @dev This function is a gateway to execute specific Wasm messages on the Perp contract.
     * It is designed for operations that modify state but do not require sending or receiving funds.
     * Examples of such operations include:
     * - PERP:
     *   - UpdateOpenLimitOrder
     *   - UpdateTp
     *   - CreateReferrerCode
     *   - RedeemReferrerCode
     * - VAULT:
     *   - CancelWithdrawRequest
     *   - UnlockDeposit
     * @param wasmMsgExecute The Wasm message to be executed, encoded as bytes.
     */
    function executeSimpleFunctions(bytes memory wasmMsgExecute) external {
        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](0);
        _performWasmExecute(perpContractAddress, wasmMsgExecute, funds);
    }

    /**
     * @notice Updates the leverage of an open position.
     * @dev This function allows updating leverage for market orders (open positions) only.
     *      When increasing leverage, excess collateral is returned to the user.
     *      When decreasing leverage, additional collateral must be provided.
     * @param tradeIndex The index of the trade to update.
     * @param newLeverage The new leverage value (must be within min/max bounds).
     * @param collateralIndex The index of the collateral token (needed to handle funds).
     * @param collateralAmount Amount of collateral to send (only for leverage decrease).
     * @param useErc20Amount Amount of ERC20 to convert to bank tokens before updating.
     */
    function updateLeverage(
        uint256 tradeIndex,
        uint256 newLeverage,
        uint256 collateralIndex,
        uint256 collateralAmount,
        uint256 useErc20Amount
    ) external {
        // Get collateral token info
        address collateralAddress = FUNTOKEN_PRECOMPILE.getErc20Address(
            getDenomOfCollateralIndex(collateralIndex)
        );

        // Handle ERC20 to bank conversion if needed
        if (useErc20Amount > 0) {
            IFunToken.NibiruAccount memory whoAddrs = FUNTOKEN_PRECOMPILE
                .whoAmI(Strings.toHexString(uint160(msg.sender), 20));

            _performSendToBank(
                collateralAddress,
                useErc20Amount,
                whoAddrs.bech32Addr
            );
        }

        // Construct UpdateLeverage message
        bytes memory wasmMsgExecute = abi.encodePacked(
            '{"update_leverage":{"trade_index":"UserTradeIndex(',
            tradeIndex.toString(),
            ')","new_leverage":"',
            newLeverage.toString(),
            '"}}'
        );

        // Prepare funds array
        INibiruEvm.BankCoin[] memory funds;
        if (collateralAmount > 0) {
            (, , IFunToken.FunToken memory token, ) = FUNTOKEN_PRECOMPILE
                .balance(msg.sender, collateralAddress);

            funds = new INibiruEvm.BankCoin[](1);
            funds[0] = INibiruEvm.BankCoin({
                denom: token.bankDenom,
                amount: collateralAmount
            });
        } else {
            funds = new INibiruEvm.BankCoin[](0);
        }

        _performWasmExecute(perpContractAddress, wasmMsgExecute, funds);
    }

    /**
     * @notice Updates both take profit and leverage in a single transaction.
     * @dev Executes two operations: first UpdateTp, then delegates to updateLeverage.
     * @param wasmMsgUpdateTp The Wasm message for updating take profit, encoded as bytes.
     * @param tradeIndex The index of the trade to update leverage for.
     * @param newLeverage The new leverage value (must be within min/max bounds).
     * @param collateralIndex The index of the collateral token (for leverage update).
     * @param collateralAmount Amount of collateral to send (only for leverage decrease).
     * @param useErc20Amount Amount of ERC20 to convert before updating leverage.
     */
    function updateTpAndLeverage(
        bytes memory wasmMsgUpdateTp,
        uint256 tradeIndex,
        uint256 newLeverage,
        uint256 collateralIndex,
        uint256 collateralAmount,
        uint256 useErc20Amount
    ) external {
        // First, execute the UpdateTp (no funds needed)
        INibiruEvm.BankCoin[] memory emptyFunds = new INibiruEvm.BankCoin[](0);
        _performWasmExecute(perpContractAddress, wasmMsgUpdateTp, emptyFunds);

        // Then update leverage using the existing function
        this.updateLeverage(
            tradeIndex,
            newLeverage,
            collateralIndex,
            collateralAmount,
            useErc20Amount
        );
    }

    // ===================== WASM EXECUTE MSG ENCODING HELPERS =====================
    // These helpers return the JSON-encoded bytes for each WASM ExecuteMsg
    function getDenomOfCollateralIndex(
        uint256 tokenIndex
    ) public view returns (string memory denom) {
        string memory jsonResponse = string(
            WASM_PRECOMPILE.query(
                perpContractAddress,
                bytes(
                    string(
                        abi.encodePacked(
                            '{"get_collateral":{"index":',
                            Strings.toString(tokenIndex),
                            "}}"
                        )
                    )
                )
            )
        );

        if (bytes(jsonResponse).length == 0) {
            revert("Response is empty");
        }

        // Find "denom": and extract the value
        bytes memory response = bytes(jsonResponse);
        uint256 denomStart = 0;

        // Search for "denom":"
        bytes memory denomKey = bytes('"denom":"');
        bool foundDenom = false;

        if (response.length >= denomKey.length) {
            for (uint256 i = 0; i <= response.length - denomKey.length; i++) {
                bool matched = true;
                for (uint256 j = 0; j < denomKey.length; j++) {
                    if (response[i + j] != denomKey[j]) {
                        matched = false;
                        break;
                    }
                }
                if (matched) {
                    denomStart = i + denomKey.length;
                    foundDenom = true;
                    break;
                }
            }
        }

        if (!foundDenom) {
            revert("Denom field not found");
        }

        // Find the closing quote
        uint256 denomEnd = denomStart;
        while (denomEnd < response.length && response[denomEnd] != '"') {
            denomEnd++;
        }
        require(
            denomEnd < response.length,
            "Denom value closing quote not found"
        );

        // Extract the denom value
        bytes memory denomBytes = new bytes(denomEnd - denomStart);
        for (uint256 i = 0; i < denomBytes.length; i++) {
            denomBytes[i] = response[denomStart + i];
        }

        return string(denomBytes);
    }

    /**
     * @notice Retrieves the collateral index (TokenIndex) for a specific trade.
     * @param userBech32Address The bech32 address of the trader.
     * @param tradeId The index of the trade.
     * @return collateralId The numerical ID of the collateral token.
     */
    function getCollateralIndexOfTrade(
        string memory userBech32Address,
        uint256 tradeId
    ) public view returns (uint256 collateralId) {
        // 1. Construct query string
        // Example: {"get_trade":{"nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul":"nibi123...","index":27}}
        string memory queryString = string(
            abi.encodePacked(
                '{"get_trade":{"nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul":"',
                userBech32Address,
                '","index":',
                tradeId.toString(), // Use .toString() for uint256
                "}}"
            )
        );

        // 2. Perform Wasm query
        string memory jsonResponse = string(
            WASM_PRECOMPILE.query(perpContractAddress, bytes(queryString))
        );

        require(
            bytes(jsonResponse).length > 0,
            "Wasm query returned empty response for get_trade"
        );

        // 3. Parse JSON response to find "collateral_index":"TokenIndex(N)"
        // Example "collateral_index":"TokenIndex(3)"
        bytes memory responseBytes = bytes(jsonResponse);
        bytes memory searchKey = bytes('"collateral_index":"TokenIndex(');
        uint256 valueStartIndex = 0;
        bool foundKey = false;

        if (responseBytes.length >= searchKey.length) {
            for (
                uint256 i = 0;
                i <= responseBytes.length - searchKey.length;
                i++
            ) {
                bool matched = true;
                for (uint256 j = 0; j < searchKey.length; j++) {
                    if (responseBytes[i + j] != searchKey[j]) {
                        matched = false;
                        break;
                    }
                }
                if (matched) {
                    valueStartIndex = i + searchKey.length; // Start of the number part
                    foundKey = true;
                    break;
                }
            }
        }

        require(
            foundKey,
            'Collateral index key (\'"collateral_index":"TokenIndex(\') not found in response'
        );

        // 4. Extract the number string
        uint256 valueEndIndex = valueStartIndex;
        while (
            valueEndIndex < responseBytes.length &&
            responseBytes[valueEndIndex] >= bytes1("0") && // Check for digit
            responseBytes[valueEndIndex] <= bytes1("9")
        ) {
            // Check for digit
            valueEndIndex++;
        }

        require(
            valueEndIndex > valueStartIndex,
            "Collateral index value is empty"
        );
        // Check for the closing parenthesis immediately after the number
        require(
            valueEndIndex < responseBytes.length &&
                responseBytes[valueEndIndex] == bytes1(")"),
            "Collateral index value malformed (missing ')' or non-numeric characters)"
        );

        bytes memory numBytes = new bytes(valueEndIndex - valueStartIndex);
        for (uint256 i = 0; i < numBytes.length; i++) {
            numBytes[i] = responseBytes[valueStartIndex + i];
        }

        // 5. Convert number string to uint256
        collateralId = _stringToUint(string(numBytes));
        return collateralId;
    }

    /**
     * @dev Converts a string representation of an unsigned integer to uint256.
     *      Reverts if the string is empty, or contains non-digit characters.
     */
    function _stringToUint(
        string memory s
    ) private pure returns (uint256 result) {
        bytes memory b = bytes(s);
        require(b.length > 0, "stringToUint: String is empty");
        result = 0;
        for (uint i = 0; i < b.length; i++) {
            uint8 charCode = uint8(b[i]);
            require(
                charCode >= 48 && charCode <= 57,
                "stringToUint: Non-digit character found"
            ); // ASCII '0' to '9'
            uint256 digit = charCode - 48;
            // Check for overflow before multiplication/addition
            require(
                result <= (type(uint256).max - digit) / 10,
                "stringToUint: Integer overflow"
            );
            result = result * 10 + digit;
        }
        return result;
    }

    function removeQuotes(
        string memory str
    ) internal pure returns (string memory) {
        bytes memory b = bytes(str);
        if (b.length >= 2 && b[0] == '"' && b[b.length - 1] == '"') {
            bytes memory result = new bytes(b.length - 2);
            for (uint i = 1; i < b.length - 1; i++) {
                result[i - 1] = b[i];
            }
            return string(result);
        }
        return str;
    }
}
