// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.8.19;

import "@nibiruchain/solidity/contracts/IFunToken.sol";
import "@nibiruchain/solidity/contracts/Wasm.sol";
import "@nibiruchain/solidity/contracts/NibiruEvmUtils.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/**
 * @title SimpleVaultInterface
 * @dev Simplified interface for vault deposits to measure gas consumption
 * This contract focuses on the deposit flow: ERC20 -> Bank tokens -> Vault deposit
 */
contract SimpleVaultInterface {
    using Strings for uint256;
    using Strings for address;

    string public vaultAddress;
    address public collateralErc20;
    string public collateralBankDenom;
    
    event DepositCompleted(
        address indexed depositor,
        uint256 erc20Amount,
        uint256 depositAmount,
        uint256 gasUsed
    );

    constructor(
        string memory _vaultAddress,
        address _collateralErc20,
        string memory _collateralBankDenom
    ) {
        vaultAddress = _vaultAddress;
        collateralErc20 = _collateralErc20;
        collateralBankDenom = _collateralBankDenom;
    }

    /**
     * @notice Simple deposit function for gas measurement
     * @dev Converts ERC20 to bank tokens and deposits into vault
     * @param erc20Amount Amount of ERC20 tokens to convert and deposit
     */
    function deposit(uint256 erc20Amount) external {
        uint256 gasStart = gasleft();
        
        // Step 1: Get the depositor's Nibiru addresses
        IFunToken.NibiruAccount memory depositorAddrs = FUNTOKEN_PRECOMPILE.whoAmI(
            msg.sender.toHexString()
        );
        
        // Step 2: Convert ERC20 to bank tokens
        uint256 bankAmount = FUNTOKEN_PRECOMPILE.sendToBank(
            collateralErc20,
            erc20Amount,
            depositorAddrs.bech32Addr
        );
        
        // Step 3: Prepare deposit message for vault
        bytes memory depositMsg = bytes('{"deposit":{}}');
        
        // Step 4: Prepare funds for the deposit
        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: collateralBankDenom,
            amount: bankAmount
        });
        
        // Step 5: Execute deposit on vault
        WASM_PRECOMPILE.execute(
            vaultAddress,
            depositMsg,
            funds
        );
        
        uint256 gasUsed = gasStart - gasleft();
        emit DepositCompleted(msg.sender, erc20Amount, bankAmount, gasUsed);
    }
    
    /**
     * @notice Measure gas for ERC20 to bank conversion only
     */
    function measureSendToBank(uint256 amount) external returns (uint256 gasUsed) {
        uint256 gasStart = gasleft();
        
        IFunToken.NibiruAccount memory depositorAddrs = FUNTOKEN_PRECOMPILE.whoAmI(
            msg.sender.toHexString()
        );
        
        FUNTOKEN_PRECOMPILE.sendToBank(
            collateralErc20,
            amount,
            depositorAddrs.bech32Addr
        );
        
        gasUsed = gasStart - gasleft();
    }
    
    /**
     * @notice Measure gas for vault deposit only (assuming bank tokens available)
     */
    function measureVaultDeposit(uint256 amount) external returns (uint256 gasUsed) {
        uint256 gasStart = gasleft();
        
        bytes memory depositMsg = bytes('{"deposit":{}}');
        
        INibiruEvm.BankCoin[] memory funds = new INibiruEvm.BankCoin[](1);
        funds[0] = INibiruEvm.BankCoin({
            denom: collateralBankDenom,
            amount: amount
        });
        
        WASM_PRECOMPILE.execute(
            vaultAddress,
            depositMsg,
            funds
        );
        
        gasUsed = gasStart - gasleft();
    }
    
    /**
     * @notice Query vault for testing
     */
    function queryVault(bytes memory queryMsg) external view returns (bytes memory) {
        return WASM_PRECOMPILE.query(vaultAddress, queryMsg);
    }
    
    /**
     * @notice Get vault's collateral denomination
     */
    function getVaultCollateralDenom() external view returns (string memory) {
        bytes memory response = WASM_PRECOMPILE.query(
            vaultAddress,
            bytes('{"get_collateral_denom":{}}')
        );
        return removeQuotes(string(response));
    }
    
    /**
     * @notice Get vault's share denomination
     */
    function getVaultShareDenom() external view returns (string memory) {
        bytes memory response = WASM_PRECOMPILE.query(
            vaultAddress,
            bytes('{"get_vault_share_denom":{}}')
        );
        return removeQuotes(string(response));
    }
    
    /**
     * @dev Helper to remove quotes from JSON response
     */
    function removeQuotes(string memory str) internal pure returns (string memory) {
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