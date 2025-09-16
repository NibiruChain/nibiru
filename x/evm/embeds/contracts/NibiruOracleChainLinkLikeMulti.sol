// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

import "./IOracle.sol";
import "@openzeppelin/contracts/utils/math/Math.sol";

/// @title NibiruOracleChainLinkMulti
/// @notice This contract serves as a ChainLink-like data feed that sources its
/// "answer" value from the Nibiru Oracle system.
///   _   _  _____  ____  _____  _____   _    _
///  | \ | ||_   _||  _ \|_   _||  __ \ | |  | |
///  |  \| |  | |  | |_) | | |  | |__) || |  | |
///  | . ` |  | |  |  _ <  | |  |  _  / | |  | |
///  | |\  | _| |_ | |_) |_| |_ | | \ \ | |__| |
///  |_| \_||_____||____/|_____||_|  \_\ \____/
///
/// The Nibiru Oracle gives price data with 18 decimals universally,
/// and that 18-decimal answer is scaled to have the number of decimals
/// specified by "decimals()". This is set at the time of deployment.
/// Decimals can only be modified in the case of an upgradeable proxy.
///
/// NibiruOracleChainLinkMulti is the multiple oracle source version of
/// NibiruOracleChainLinkLike, which uses one oracle pair as its first argument.
///
/// Oracle pairs are any of the valid arguments for the gRPC query,
/// "/nibiru.oracle.v1.Query/ExchangeRate". The current acive set of oracle
/// pairs can be retrieved with the "/nibiru.oracle.v1.Query/Actives" query.
/// With the Nibiru CLI, that's `nibid oracle query actives | jq`
contract NibiruOracleChainLinkMulti is ChainLinkAggregatorV3Interface {
    using Math for uint256;

    string[] public oracles; // ordered pairs
    uint8 public _decimals; // use the decimals() query
    string public name;

    uint256 constant MAX_ORACLES = 4;

    /// @notice Guards against returning a zero value for price.
    error NegativeOracle(int256 value);
    error ZeroOracle();

    constructor(string[] memory _oracles, uint8 _dec, string memory _name) {
        require(
            _dec <= 18,
            "NibiruOracleChainLinkMulti: Decimals cannot exceed 18"
        );
        require(
            _oracles.length > 1,
            "NibiruOracleChainLinkMulti: Too few oracles"
        );
        require(
            _oracles.length <= MAX_ORACLES,
            string.concat(
                "NibiruOracleChainLinkMulti: Max number of multiplications ",
                "exceeded (4); too many oracle pairs given"
            )
        );
        name = _name;
        oracles = _oracles;
        _decimals = _dec;
    }

    function decimals() external view override returns (uint8) {
        return _decimals;
    }

    /// @notice Returns a human-readable description of the oracle.
    function description() external view override returns (string memory) {
        return
            string.concat(
                "Nibiru Oracle ChainLink-like composite feed (multi) for: ",
                name
            );
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
        // Seed with first leg
        (
            uint80 r0,
            int256 a0,
            uint256 s0,
            uint256 u0,
            uint80 ar0
        ) = NIBIRU_ORACLE.chainLinkLatestRoundData(oracles[0]);
        if (a0 < 0) revert NegativeOracle(a0);

        uint256 acc = uint256(a0); // 18-dec accumulator (unsigned)
        roundId = r0; // max across legs
        answeredInRound = ar0; // max across legs
        startedAt = s0; // min across legs
        updatedAt = u0; // min across legs

        // Fold remaining legs: acc = acc * ai / 1e18
        for (uint256 i = 1; i < oracles.length; ++i) {
            (
                uint80 ri,
                int256 ai,
                uint256 si,
                uint256 ui,
                uint80 ari
            ) = NIBIRU_ORACLE.chainLinkLatestRoundData(oracles[i]);
            if (ai == 0) revert ZeroOracle();
            if (ai < 0) revert NegativeOracle(ai);

            acc = acc.mulDiv(uint256(ai), 1e18); // full-precision, trunc toward zero

            if (ri > roundId) roundId = ri;
            if (ari > answeredInRound) answeredInRound = ari;
            if (si < startedAt) startedAt = si;
            if (ui < updatedAt) updatedAt = ui;
        }

        // Scale 18 -> _decimals (truncate)
        if (_decimals < 18) {
            acc = acc / (10 ** (18 - _decimals));
        }
        answer = int256(acc);

        return (roundId, answer, startedAt, updatedAt, answeredInRound);
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

    function latestAnswer() public view returns (int256) {
        (, int256 answer, , , ) = latestRoundData();
        return answer;
    }

    /// @notice Returns the Unix timestamp in seconds for the block of the latest
    /// answer for the oracle pair.
    function latestTimestamp() public view returns (uint256) {
        (, , , uint256 updatedAt, ) = latestRoundData();
        return updatedAt;
    }

    function latestRound() external view override returns (uint256) {
        (uint80 roundId, , , , ) = latestRoundData();
        return uint256(roundId);
    }

    /// @notice Returns the latest answer from the Nibiru Oracle. Historical round
    /// retrieval is not supported. This method is a duplicate of
    /// "latestAnswer".
    function getAnswer(uint256) external view returns (int256) {
        return latestAnswer();
    }

    /// @notice Returns the Unix timestamp in seconds for the block of the latest
    /// answer. This method is a duplicate of "latestTimestamp".
    function getTimestamp(uint256) external view returns (uint256) {
        return latestTimestamp();
    }
}
