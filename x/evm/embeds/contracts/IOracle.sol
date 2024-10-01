// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @dev Oracle interface for querying exchange rates
interface IOracle {
  /// @dev queryExchangeRate queries the exchange rate for a given pair
  /// @param pair the pair to query
  function queryExchangeRate(string memory pair) external;
}

address constant ORACLE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000801;

IOracle constant ORACLE_GATEWAY = IOracle(ORACLE_PRECOMPILE_ADDRESS);
