import type { HexAddr } from "./precompile"

/** Address of Nibiru's Oracle precompiled contract */
export const ADDR_ORACLE_PRECOMPILE: HexAddr =
  "0x0000000000000000000000000000000000000801"

/** Contract ABI for Nibiru's Oracle precompiled contract.  */
export const ABI_ORACLE_PRECOMPILE = [
  {
    inputs: [
      {
        internalType: "string",
        name: "pair",
        type: "string",
      },
    ],
    name: "chainLinkLatestRoundData",
    outputs: [
      {
        internalType: "uint80",
        name: "roundId",
        type: "uint80",
      },
      {
        internalType: "int256",
        name: "answer",
        type: "int256",
      },
      {
        internalType: "uint256",
        name: "startedAt",
        type: "uint256",
      },
      {
        internalType: "uint256",
        name: "updatedAt",
        type: "uint256",
      },
      {
        internalType: "uint80",
        name: "answeredInRound",
        type: "uint80",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "pair",
        type: "string",
      },
    ],
    name: "queryExchangeRate",
    outputs: [
      {
        internalType: "uint256",
        name: "price",
        type: "uint256",
      },
      {
        internalType: "uint64",
        name: "blockTimeMs",
        type: "uint64",
      },
      {
        internalType: "uint64",
        name: "blockHeight",
        type: "uint64",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
] as const
