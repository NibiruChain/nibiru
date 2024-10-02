// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @notice Oracle interface for querying exchange rates
interface IOracle {
  /// @notice Queries the exchange rate for a given pair
  /// @param pair The asset pair to query. For example, "ubtc:uusd" is the 
  /// USD price of BTC and "unibi:uusd" is the USD price of NIBI.
  /// @return The exchange rate (a decimal value) as a string.
  /// @dev This function is view-only and does not modify state.
  function queryExchangeRate(string memory pair) external view returns (string memory);
}

address constant ORACLE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000801;

IOracle constant ORACLE_GATEWAY = IOracle(ORACLE_PRECOMPILE_ADDRESS);
