// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

import "./IOracle.sol";

/// @title NibiruOracleChainLinkLike
/// @notice This contract serves as a ChainLink-like data feed that sources its
/// "answer" value from the Nibiru Oracle system. The Nibiru Oracle gives price
/// data with 18 decimals universally, and that 18-decimal answer is scaled to
/// have the number of decimals specified by "decimals()". This is set at the
/// time of deployment.
///   _   _  _____  ____  _____  _____   _    _
///  | \ | ||_   _||  _ \|_   _||  __ \ | |  | |
///  |  \| |  | |  | |_) | | |  | |__) || |  | |
///  | . ` |  | |  |  _ <  | |  |  _  / | |  | |
///  | |\  | _| |_ | |_) |_| |_ | | \ \ | |__| |
///  |_| \_||_____||____/|_____||_|  \_\ \____/
///
contract NibiruOracleChainLinkLike is ChainLinkAggregatorV3Interface {
    string public pair;
    uint8 public _decimals;

    constructor(string memory _pair, uint8 _dec) {
        require(_dec <= 18, "Decimals cannot exceed 18");
        require(bytes(_pair).length > 0, "Pair string cannot be empty");
        pair = _pair;
        _decimals = _dec;
    }

    function decimals() external view override returns (uint8) {
        return _decimals;
    }

    /// @notice Returns a human-readable description of the oracle and its data
    /// feed identifier (pair) in the Nibiru Oracle system
    function description() external view override returns (string memory) {
        return
            string.concat("Nibiru Oracle ChainLink-like price feed for ", pair);
    }

    /// @notice Oracle version number. Hardcoded to 1.
    function version() external pure override returns (uint256) {
        return 1;
    }

    /// @notice Returns the latest data from the Nibiru Oracle.
    /// @return roundId The block number when the answer was published onchain.
    /// @return answer Data feed result scaled to the precision specified by
    ///   "decimals()"
    /// @return startedAt UNIX timestamp in seconds when "answer" was published.
    /// @return updatedAt UNIX timestamp in seconds when "answer" was published.
    /// @return answeredInRound The ID of the round where the answer was computed.
    ///   Since the Nibiru Oracle does not have ChainLink's system of voting
    ///   rounds, this argument is a meaningless, arbitrary constant.
    function latestRoundData()
        public
        view
        override
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        (
            uint80 _roundId,
            int256 answer18Dec,
            uint256 _startedAt,
            uint256 _updatedAt,
            uint80 _answeredInRound
        ) = NIBIRU_ORACLE.chainLinkLatestRoundData(pair);
        answer = scaleAnswerToDecimals(answer18Dec);
        return (_roundId, answer, _startedAt, _updatedAt, _answeredInRound);
    }

    /// @notice Returns the latest data from the Nibiru Oracle. Historical round
    /// retrieval is not supported. This method is a duplicate of
    /// "latestRoundData".
    /// @return roundId The block number when the answer was published onchain.
    /// @return answer Data feed result scaled to the precision specified by
    ///   "decimals()"
    /// @return startedAt UNIX timestamp in seconds when "answer" was published.
    /// @return updatedAt UNIX timestamp in seconds when "answer" was published.
    /// @return answeredInRound The ID of the round where the answer was computed.
    ///   Since the Nibiru Oracle does not have ChainLink's system of voting
    ///   rounds, this argument is a meaningless, arbitrary constant.
    function getRoundData(
        uint80
    )
        external
        view
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return latestRoundData();
    }

    function scaleAnswerToDecimals(
        int256 answer18Dec
    ) internal view returns (int256 answer) {
        // Default answers are in 18 decimals.
        // Scale down to the decimals specified in the constructor.
        uint8 pow10 = 18 - _decimals;
        return answer18Dec / int256(10 ** pow10);
    }
}
