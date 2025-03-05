import { type ContractRunner, Contract, type InterfaceAbi } from "ethers"
import {
  ADDR_WASM_PRECOMPILE,
  ADDR_ORACLE_PRECOMPILE,
  ADDR_FUNTOKEN_PRECOMPILE,
  ABI_WASM_PRECOMPILE,
  ABI_ORACLE_PRECOMPILE,
  ABI_FUNTOKEN_PRECOMPILE,
} from "../precompile"

export const ETHERS_ABI = {
  WASM: ABI_WASM_PRECOMPILE as InterfaceAbi,
  ORACLE: ABI_ORACLE_PRECOMPILE as InterfaceAbi,
  FUNTOKEN: ABI_FUNTOKEN_PRECOMPILE as InterfaceAbi,
}

import {
  type IWasm,
  type IOracle,
  type IFunToken,
  type NibiruOracleChainLinkLike,
  type ERC20Minter,
  NibiruOracleChainLinkLike__factory,
  ERC20Minter__factory,
} from "./typechain"

export const wasmPrecompile = (runner: ContractRunner): IWasm =>
  new Contract(
    ADDR_WASM_PRECOMPILE,
    ETHERS_ABI.WASM,
    runner,
  ) as unknown as IWasm

export const oraclePrecompile = (runner: ContractRunner): IOracle =>
  new Contract(
    ADDR_ORACLE_PRECOMPILE,
    ETHERS_ABI.ORACLE,
    runner,
  ) as unknown as IOracle

export const funtokenPrecompile = (runner: ContractRunner): IFunToken =>
  new Contract(
    ADDR_FUNTOKEN_PRECOMPILE,
    ETHERS_ABI.FUNTOKEN,
    runner,
  ) as unknown as IFunToken

/**
 * Returns a typed contract instance for one of the NibiruOracleChainLinkLike.sol
 * contracts. These implement ChainLink's interface, AggregatorV3Interface.sol, but source
 * data from the Nibiru Oracle mechanism, which publishes data much faster than
 * ChainLink.
 * */
export const chainlinkLike = (
  runner: ContractRunner,
  addr: string,
): NibiruOracleChainLinkLike =>
  NibiruOracleChainLinkLike__factory.connect(addr, runner)

/**
 * Returns a typed contract instance for a standard ERC20 contract.
 * */
export const erc20Caller = (
  runner: ContractRunner,
  addr: string,
): ERC20Minter => ERC20Minter__factory.connect(addr, runner)
