import { describe, expect, it, jest } from '@jest/globals';
import { ethers, TransactionReceipt, Log } from 'ethers';
import { account } from './setup';
import { deployContractEventsEmitter, deployContractTransactionReverter, TENPOW12 } from './utils';
import { TestERC20__factory } from '../types';

describe('Transaction Receipt Tests', () => {
  jest.setTimeout(15e3);

  let recipient = ethers.Wallet.createRandom().address;

  it('simple transfer receipt', async () => {
    const value = ethers.parseEther('0.0001');
    const tx = await account.sendTransaction({
      to: recipient,
      value,
    });
    const receipt = await tx.wait();

    assertBaseReceiptFields(receipt);
    expect(receipt.to).toEqual(recipient);
    expect(receipt.logs).toHaveLength(0); // ETH transfers have no logs
  });

  it('contract deployment receipt', async () => {
    const factory = new TestERC20__factory(account);
    const deployTx = await factory.deploy({ maxFeePerGas: TENPOW12 });
    const receipt = await deployTx.deploymentTransaction().wait();

    assertBaseReceiptFields(receipt);
    expect(receipt.to).toBeNull(); // Contract creation has no 'to' address
    expect(receipt.contractAddress).toBeDefined();

    // Verify the deployed contract address is valid
    const code = await account.provider.getCode(receipt.contractAddress!);
    expect(code).not.toEqual('0x');
  });

  it('receipt with logs / events', async () => {
    const contract = await deployContractEventsEmitter();
    const expectedValue = 123n;

    const tx = await contract.emitEvent(expectedValue);
    const receipt = await tx.wait();

    assertBaseReceiptFields(receipt);
    expect(receipt.to).toEqual(contract.target.toString());

    // Event specific checks
    expect(receipt.logs.length).toEqual(1);
    const event = receipt.logs[0];
    assertEventLogFields(event, contract.target.toString());

    // Event data checks
    const eventSignature = 'TestEvent(address,uint256)';
    expect(event.topics[0]).toEqual(ethers.id(eventSignature));
    expect(event.topics.length).toEqual(2); // topic[0] is hash, topic[1] is indexed param
    expect(event['args'].sender).toEqual(account.address);
    expect(event['args'].value).toEqual(expectedValue);

    // Verify indexed parameter encoding
    expect(event.topics[1]).toEqual(ethers.zeroPadValue(account.address, 32)); // indexed address param
  });
});

function assertBaseReceiptFields(receipt: TransactionReceipt) {
  // Basic transaction info
  expect(receipt.status).toEqual(1);
  expect([0, 1, 2]).toContain(receipt.type);
  expect(receipt.hash).toMatch(/^0x[a-fA-F0-9]{64}$/);
  expect(typeof receipt.index).toBe('number');

  // Block info
  expect(typeof receipt.blockNumber).toBe('number');
  expect(receipt.blockNumber).toBeGreaterThan(0);
  expect(receipt.blockHash).toMatch(/^0x[a-fA-F0-9]{64}$/);

  // Address fields
  expect(receipt.from).toEqual(account.address);
  expect(receipt.from).toMatch(/^0x[a-fA-F0-9]{40}$/);

  // Gas fields
  expect(typeof receipt.gasUsed).toBe('bigint');
  expect(receipt.gasUsed).toBeGreaterThan(0n);
  expect(typeof receipt.cumulativeGasUsed).toBe('bigint');
  expect(receipt.cumulativeGasUsed).toBeGreaterThanOrEqual(receipt.gasUsed);

  // Logs
  expect(receipt.logsBloom).toMatch(/^0x[a-fA-F0-9]{512}$/);
  expect(Array.isArray(receipt.logs)).toBeTruthy();

  // Optional fields
  if (receipt.to !== null) {
    expect(receipt.to).toMatch(/^0x[a-fA-F0-9]{40}$/);
  }
  if (receipt.contractAddress !== null) {
    expect(receipt.contractAddress).toMatch(/^0x[a-fA-F0-9]{40}$/);
  }
}

function assertEventLogFields(log: Log, expectedContractAddress: string) {
  // Block info
  expect(typeof log.blockNumber).toBe('number');
  expect(log.blockNumber).toBeGreaterThan(0);
  expect(log.blockHash).toMatch(/^0x[a-fA-F0-9]{64}$/);

  // Transaction info
  expect(typeof log.index).toBe('number');
  expect(log.transactionHash).toMatch(/^0x[a-fA-F0-9]{64}$/);

  // Address validation
  expect(log.address).toEqual(expectedContractAddress);
  expect(log.address).toMatch(/^0x[a-fA-F0-9]{40}$/);

  // Topics validation
  expect(Array.isArray(log.topics)).toBeTruthy();
  log.topics.forEach((topic) => {
    expect(topic).toMatch(/^0x[a-fA-F0-9]{64}$/);
  });

  // Data field
  expect(log.data).toMatch(/^0x([a-fA-F0-9]{2})*$/);
}
