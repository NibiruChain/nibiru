import { expect, test } from '@jest/globals';
import { toBigInt } from 'ethers';
import { deployContractNibiruOracleChainLinkLike } from './utils';
import { TEST_TIMEOUT } from './setup';

test(
  'NibiruOracleChainLinkLike implements ChainLink AggregatorV3Interface',
  async () => {
    const { oraclePair, contract } = await deployContractNibiruOracleChainLinkLike();

    const oracleAddr = await contract.getAddress();
    expect(oracleAddr).not.toBeFalsy();

    const decimals = await contract.decimals();
    expect(decimals).toEqual(BigInt(8));

    const description = await contract.description();
    expect(description).toEqual(`Nibiru Oracle ChainLink-like price feed for ${oraclePair}`);

    const version = await contract.version();
    expect(version).toEqual(1n);

    // latestRoundData
    const genesisEthUsdPrice = 2000n;
    {
      const { roundId, answer, startedAt, updatedAt, answeredInRound } = await contract.latestRoundData();
      expect(roundId).toEqual(0n); // price is from genesis block
      expect(startedAt).toBeGreaterThan(1n);
      expect(updatedAt).toBeGreaterThan(1n);
      expect(answeredInRound).toEqual(420n);
      expect(answer).toEqual(genesisEthUsdPrice * toBigInt(1e8));
    }

    // getRoundData
    {
      const { roundId, answer, startedAt, updatedAt, answeredInRound } = await contract.getRoundData(0n);
      expect(roundId).toEqual(0n); // price is from genesis block
      expect(startedAt).toBeGreaterThan(1n);
      expect(updatedAt).toBeGreaterThan(1n);
      expect(answeredInRound).toEqual(420n);
      expect(answer).toEqual(genesisEthUsdPrice * toBigInt(1e8));
    }
  },
  TEST_TIMEOUT,
);
