// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @notice Oracle interface for querying exchange rates
interface IOracle {
    /// @notice Queries the dated exchange rate for a given pair
    /// @param pair The asset pair to query. For example, "ubtc:uusd" is the
    /// USD price of BTC and "unibi:uusd" is the USD price of NIBI.
    /// @return price The exchange rate for the given pair
    /// @return blockTimeMs The block time in milliseconds when the price was
    /// last updated
    /// @return blockHeight The block height when the price was last updated
    /// @dev This function is view-only and does not modify state.
    function queryExchangeRate(
        string memory pair
    )
        external
        view
        returns (uint256 price, uint64 blockTimeMs, uint64 blockHeight);
}

// solhint-disable-next-line interface-starts-with-i
interface IChainlink {
    function decimals() external view returns (uint8);

    function description() external view returns (string memory);

    function version() external view returns (uint256);

    function getRoundData(
        uint80 _roundId
    )
        external
        view
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );

    function latestRoundData()
        external
        view
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );
}

address constant ORACLE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000801;

IOracle constant ORACLE_GATEWAY = IOracle(ORACLE_PRECOMPILE_ADDRESS);
