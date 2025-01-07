// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@chainlink/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol";
import "./IOracle.sol";

contract OracleChainLinkLike is AggregatorV3Interface {
    string public pair;

    constructor(string memory _pair) {
        pair = _pair;
    }

    function decimals() external pure override returns (uint8) {
        return 8; // Adjust as needed
    }

    function description() external view override returns (string memory) {
        return
            string(
                abi.encodePacked(
                    "Nibiru Oracle ChainLink-like price feed for ",
                    pair
                )
            );
    }

    function version() external pure override returns (uint256) {
        return 1;
    }

    function latestRoundData()
        external
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
        return ORACLE_GATEWAY.latestRoundData(pair);
    }

    function getRoundData(
        uint80
    )
        external
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
        return ORACLE_GATEWAY.latestRoundData(pair);
    }
}
