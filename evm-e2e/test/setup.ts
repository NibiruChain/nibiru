import { config } from 'dotenv';
import { ethers, getDefaultProvider, Wallet } from 'ethers';

config();

const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT);
const account = Wallet.fromPhrase(process.env.MNEMONIC, provider);
const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || 15000;
const TX_WAIT_TIMEOUT = Number(process.env.TX_WAIT_TIMEOUT) || 5000;

export { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT };
