import {
  funtokenPrecompile,
  wasmPrecompile,
} from "@nibiruchain/evm-core/ethers";
import {
  ADDR_WASM_PRECOMPILE,
  ADDR_FUNTOKEN_PRECOMPILE,
  ADDR_ORACLE_PRECOMPILE,
  ABI_WASM_PRECOMPILE,
  ABI_FUNTOKEN_PRECOMPILE,
  ABI_ORACLE_PRECOMPILE,
} from "@nibiruchain/evm-core";
import { config } from "dotenv";
import { ethers, Wallet } from "ethers";

const setupProvider = () => {
  config();
  const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT);
  const account = Wallet.fromPhrase(process.env.MNEMONIC!, provider);
  console.log("✅ Setup succeeded");
  return { provider, account };
};

const ethersExamples = async () => {
  const { provider, account } = setupProvider();
  // NOTE: Both wallets and providers are valid ethers.ContractRunner instances,
  // meaning both `account` and `provider` local variables make sense.
  let wasmCaller = wasmPrecompile(account);
  wasmCaller = wasmPrecompile(provider);

  wasmCaller.execute;
  wasmCaller.executeMulti;
  wasmCaller.query;
  wasmCaller.queryRaw;
  wasmCaller.execute;

  const funtokenCaller = funtokenPrecompile(account);
  const whoAddr = await account.getAddress();
  const resp = await funtokenCaller.whoAmI(whoAddr);
  console.debug("DEBUG %o: ", { resp });
  const [addrHex, addrBech32] = resp as unknown as [string, string];
  console.debug("DEBUG %o: ", { addrHex, addrBech32 });

  const respBankBalance = await funtokenCaller.bankBalance(addrHex, "unibi");
  console.debug("DEBUG %o: ", { respBankBalance });

  funtokenCaller.balance;
  funtokenCaller.bankMsgSend;
  funtokenCaller.sendToEvm;
  funtokenCaller.sendToBank;
};

const main = async () => {
  await ethersExamples();
};

main();
