// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

import "./IOracle.sol";

/// @notice MIM/Abracadabra FeeHandler-shaped fixture for the 0x0801 oracle
/// compatibility path. The deployed Nibiru FeeHandler at
/// https://nibiscan.io/address/0x279D54aDD72935d845074675De0dbcfdc66800a3/contract/6900/code
/// uses this no-arg `quoteNativeFee()` interface to quote the protocol native
/// fee from the Nibiru oracle precompile.
contract TestOracleAsLZNativeFeeHandler {
    uint256 public constant DEFAULT_USD_FEE = 1e18;
    string public constant NATIVE_PAIR = "unibi:uusd";

    IOracle public oracle;

    constructor(address oracle_) {
        oracle = IOracle(oracle_);
    }

    function quoteNativeFee() external view returns (uint256) {
        (uint256 price, , ) = oracle.queryExchangeRate(NATIVE_PAIR);
        require(price > 0, "invalid price");
        return 1e18 * DEFAULT_USD_FEE / price;
    }
}
